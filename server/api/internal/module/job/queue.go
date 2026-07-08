package job

import "context"

type CropJobQueue interface {
	EnqueueCropJob(ctx context.Context, cropJob CropJob) error
}
