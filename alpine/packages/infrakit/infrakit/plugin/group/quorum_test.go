package group

import (
	mock_group "github.com/docker/infrakit/mock/plugin/group"
	"github.com/docker/infrakit/spi/instance"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

var (
	a = instance.Description{ID: instance.ID("a"), LogicalID: logicalID("one")}
	b = instance.Description{ID: instance.ID("b"), LogicalID: logicalID("two")}
	c = instance.Description{ID: instance.ID("c"), LogicalID: logicalID("three")}
	d = instance.Description{ID: instance.ID("d"), LogicalID: logicalID("four")}

	logicalIDs = []instance.LogicalID{
		*a.LogicalID,
		*b.LogicalID,
		*c.LogicalID,
	}
)

func logicalID(value string) *instance.LogicalID {
	id := instance.LogicalID(value)
	return &id
}

func TestQuorumOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	scaled := mock_group.NewMockScaled(ctrl)
	quorum := NewQuorum(scaled, logicalIDs, 1*time.Millisecond)

	gomock.InOrder(
		scaled.EXPECT().List().Return([]instance.Description{a, b, c}, nil),
		scaled.EXPECT().List().Do(func() {
			go quorum.Stop()
		}).Return([]instance.Description{a, b, c}, nil),
		// Allow subsequent calls to List() to mitigate ordering flakiness of async Stop() call.
		scaled.EXPECT().List().Return([]instance.Description{a, b, c}, nil).AnyTimes(),
	)

	quorum.Run()
}

func TestRestoreQuorum(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	scaled := mock_group.NewMockScaled(ctrl)
	quorum := NewQuorum(scaled, logicalIDs, 1*time.Millisecond)

	logicalID := *c.LogicalID
	gomock.InOrder(
		scaled.EXPECT().List().Return([]instance.Description{a, b, c}, nil),
		scaled.EXPECT().List().Return([]instance.Description{a, b, c}, nil),
		scaled.EXPECT().List().Return([]instance.Description{a, b}, nil),
		scaled.EXPECT().CreateOne(&logicalID),
		scaled.EXPECT().List().Do(func() {
			go quorum.Stop()
		}).Return([]instance.Description{a, b, c}, nil),
		// Allow subsequent calls to List() to mitigate ordering flakiness of async Stop() call.
		scaled.EXPECT().List().Return([]instance.Description{a, b, c}, nil).AnyTimes(),
	)

	quorum.Run()
}

func TestRemoveUnknown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	scaled := mock_group.NewMockScaled(ctrl)
	quorum := NewQuorum(scaled, logicalIDs, 1*time.Millisecond)

	gomock.InOrder(
		scaled.EXPECT().List().Return([]instance.Description{a, c, b}, nil),
		scaled.EXPECT().List().Return([]instance.Description{c, a, d, b}, nil),
		scaled.EXPECT().List().Do(func() {
			go quorum.Stop()
		}).Return([]instance.Description{a, b, c}, nil),
		// Allow subsequent calls to List() to mitigate ordering flakiness of async Stop() call.
		scaled.EXPECT().List().Return([]instance.Description{a, b, c}, nil).AnyTimes(),
	)

	scaled.EXPECT().Destroy(d.ID)

	quorum.Run()
}
