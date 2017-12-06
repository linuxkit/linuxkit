package images

import (
	gocontext "context"

	"github.com/boltdb/bolt"
	eventstypes "github.com/containerd/containerd/api/events"
	imagesapi "github.com/containerd/containerd/api/services/images/v1"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/events"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/metadata"
	"github.com/containerd/containerd/plugin"
	ptypes "github.com/gogo/protobuf/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.GRPCPlugin,
		ID:   "images",
		Requires: []plugin.Type{
			plugin.MetadataPlugin,
			plugin.GCPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			m, err := ic.Get(plugin.MetadataPlugin)
			if err != nil {
				return nil, err
			}
			g, err := ic.Get(plugin.GCPlugin)
			if err != nil {
				return nil, err
			}

			return NewService(m.(*metadata.DB), ic.Events, g.(gcScheduler)), nil
		},
	})
}

type gcScheduler interface {
	ScheduleAndWait(gocontext.Context) (metadata.GCStats, error)
}

type service struct {
	db        *metadata.DB
	gc        gcScheduler
	publisher events.Publisher
}

// NewService returns the GRPC image server
func NewService(db *metadata.DB, publisher events.Publisher, gc gcScheduler) imagesapi.ImagesServer {
	return &service{
		db:        db,
		gc:        gc,
		publisher: publisher,
	}
}

func (s *service) Register(server *grpc.Server) error {
	imagesapi.RegisterImagesServer(server, s)
	return nil
}

func (s *service) Get(ctx context.Context, req *imagesapi.GetImageRequest) (*imagesapi.GetImageResponse, error) {
	var resp imagesapi.GetImageResponse

	return &resp, errdefs.ToGRPC(s.withStoreView(ctx, func(ctx context.Context, store images.Store) error {
		image, err := store.Get(ctx, req.Name)
		if err != nil {
			return err
		}
		imagepb := imageToProto(&image)
		resp.Image = &imagepb
		return nil
	}))
}

func (s *service) List(ctx context.Context, req *imagesapi.ListImagesRequest) (*imagesapi.ListImagesResponse, error) {
	var resp imagesapi.ListImagesResponse

	return &resp, errdefs.ToGRPC(s.withStoreView(ctx, func(ctx context.Context, store images.Store) error {
		images, err := store.List(ctx, req.Filters...)
		if err != nil {
			return err
		}

		resp.Images = imagesToProto(images)
		return nil
	}))
}

func (s *service) Create(ctx context.Context, req *imagesapi.CreateImageRequest) (*imagesapi.CreateImageResponse, error) {
	if req.Image.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Image.Name required")
	}

	var (
		image = imageFromProto(&req.Image)
		resp  imagesapi.CreateImageResponse
	)
	if err := s.withStoreUpdate(ctx, func(ctx context.Context, store images.Store) error {
		created, err := store.Create(ctx, image)
		if err != nil {
			return err
		}

		resp.Image = imageToProto(&created)
		return nil
	}); err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	if err := s.publisher.Publish(ctx, "/images/create", &eventstypes.ImageCreate{
		Name:   resp.Image.Name,
		Labels: resp.Image.Labels,
	}); err != nil {
		return nil, err
	}

	return &resp, nil

}

func (s *service) Update(ctx context.Context, req *imagesapi.UpdateImageRequest) (*imagesapi.UpdateImageResponse, error) {
	if req.Image.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Image.Name required")
	}

	var (
		image = imageFromProto(&req.Image)
		resp  imagesapi.UpdateImageResponse
	)
	if err := s.withStoreUpdate(ctx, func(ctx context.Context, store images.Store) error {
		var fieldpaths []string
		if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
			for _, path := range req.UpdateMask.Paths {
				fieldpaths = append(fieldpaths, path)
			}
		}

		updated, err := store.Update(ctx, image, fieldpaths...)
		if err != nil {
			return err
		}

		resp.Image = imageToProto(&updated)
		return nil
	}); err != nil {
		return nil, errdefs.ToGRPC(err)
	}

	if err := s.publisher.Publish(ctx, "/images/update", &eventstypes.ImageUpdate{
		Name:   resp.Image.Name,
		Labels: resp.Image.Labels,
	}); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *service) Delete(ctx context.Context, req *imagesapi.DeleteImageRequest) (*ptypes.Empty, error) {
	if err := s.withStoreUpdate(ctx, func(ctx context.Context, store images.Store) error {
		return errdefs.ToGRPC(store.Delete(ctx, req.Name))
	}); err != nil {
		return nil, err
	}

	if err := s.publisher.Publish(ctx, "/images/delete", &eventstypes.ImageDelete{
		Name: req.Name,
	}); err != nil {
		return nil, err
	}

	if req.Sync {
		if _, err := s.gc.ScheduleAndWait(ctx); err != nil {
			return nil, err
		}
	}

	return &ptypes.Empty{}, nil
}

func (s *service) withStore(ctx context.Context, fn func(ctx context.Context, store images.Store) error) func(tx *bolt.Tx) error {
	return func(tx *bolt.Tx) error { return fn(ctx, metadata.NewImageStore(tx)) }
}

func (s *service) withStoreView(ctx context.Context, fn func(ctx context.Context, store images.Store) error) error {
	return s.db.View(s.withStore(ctx, fn))
}

func (s *service) withStoreUpdate(ctx context.Context, fn func(ctx context.Context, store images.Store) error) error {
	return s.db.Update(s.withStore(ctx, fn))
}
