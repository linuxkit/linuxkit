package group

import (
	mock_group "github.com/docker/infrakit/mock/plugin/group"
	"github.com/docker/infrakit/spi/instance"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func TestScaleUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	scaled := mock_group.NewMockScaled(ctrl)
	scaler := NewScalingGroup(scaled, 3, 1*time.Millisecond)

	gomock.InOrder(
		scaled.EXPECT().List().Return([]instance.Description{a, b, c}, nil),
		scaled.EXPECT().List().Return([]instance.Description{a, b, c}, nil),
		scaled.EXPECT().List().Return([]instance.Description{a, b}, nil),
		scaled.EXPECT().CreateOne(nil).Return(),
		scaled.EXPECT().List().Do(func() {
			go scaler.Stop()
		}).Return([]instance.Description{a, b, c}, nil),
		// Allow subsequent calls to DescribeInstances() to mitigate ordering flakiness of async Stop() call.
		scaled.EXPECT().List().Return([]instance.Description{a, b, c, d}, nil).AnyTimes(),
	)

	scaler.Run()
}

func TestScaleDown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	scaled := mock_group.NewMockScaled(ctrl)
	scaler := NewScalingGroup(scaled, 2, 1*time.Millisecond)

	gomock.InOrder(
		scaled.EXPECT().List().Return([]instance.Description{c, b}, nil),
		scaled.EXPECT().List().Return([]instance.Description{c, a, d, b}, nil),
		scaled.EXPECT().List().Do(func() {
			go scaler.Stop()
		}).Return([]instance.Description{a, b}, nil),
		// Allow subsequent calls to DescribeInstances() to mitigate ordering flakiness of async Stop() call.
		scaled.EXPECT().List().Return([]instance.Description{c, d}, nil).AnyTimes(),
	)

	scaled.EXPECT().Destroy(a.ID)
	scaled.EXPECT().Destroy(b.ID)

	scaler.Run()
}
