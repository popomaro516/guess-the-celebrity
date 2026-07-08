package job

type Crop struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type CropJob struct {
	QuizID         string `json:"quiz_id"`
	SourceImageKey string `json:"source_image_key"`
	OutputImageKey string `json:"output_image_key"`
	Crop           Crop   `json:"crop"`
}
