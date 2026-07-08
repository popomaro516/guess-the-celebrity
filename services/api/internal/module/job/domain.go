package job

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
