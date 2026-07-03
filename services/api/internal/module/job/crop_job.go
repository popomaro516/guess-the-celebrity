package job

import "context"

type Crop struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

type CropJob struct {
	QuizID         string
	SourceImageKey string
	OutputImageKey string
	Crop           Crop
}

type CropJobQueue interface {
	EnqueueCropJob(ctx context.Context, cropJob CropJob) error
}
