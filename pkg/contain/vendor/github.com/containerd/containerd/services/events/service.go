package events

import (
	api "github.com/containerd/containerd/api/services/events/v1"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/events"
	"github.com/containerd/containerd/events/exchange"
	"github.com/containerd/containerd/plugin"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.GRPCPlugin,
		ID:   "events",
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			return NewService(ic.Events), nil
		},
	})
}

type service struct {
	events *exchange.Exchange
}

// NewService returns the GRPC events server
func NewService(events *exchange.Exchange) api.EventsServer {
	return &service{events: events}
}

func (s *service) Register(server *grpc.Server) error {
	api.RegisterEventsServer(server, s)
	return nil
}

func (s *service) Publish(ctx context.Context, r *api.PublishRequest) (*ptypes.Empty, error) {
	if err := s.events.Publish(ctx, r.Topic, r.Event); err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &ptypes.Empty{}, nil
}

func (s *service) Forward(ctx context.Context, r *api.ForwardRequest) (*ptypes.Empty, error) {
	if err := s.events.Forward(ctx, fromProto(r.Envelope)); err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	return &ptypes.Empty{}, nil
}

func (s *service) Subscribe(req *api.SubscribeRequest, srv api.Events_SubscribeServer) error {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	eventq, errq := s.events.Subscribe(ctx, req.Filters...)
	for {
		select {
		case ev := <-eventq:
			if err := srv.Send(toProto(ev)); err != nil {
				return errors.Wrapf(err, "failed sending event to subscriber")
			}
		case err := <-errq:
			if err != nil {
				return errors.Wrapf(err, "subscription error")
			}

			return nil
		}
	}
}

func toProto(env *events.Envelope) *api.Envelope {
	return &api.Envelope{
		Timestamp: env.Timestamp,
		Namespace: env.Namespace,
		Topic:     env.Topic,
		Event:     env.Event,
	}
}

func fromProto(env *api.Envelope) *events.Envelope {
	return &events.Envelope{
		Timestamp: env.Timestamp,
		Namespace: env.Namespace,
		Topic:     env.Topic,
		Event:     env.Event,
	}
}
