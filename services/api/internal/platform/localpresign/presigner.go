package localpresign

import (
	"context"
	"net/url"
	"time"
)

type Presigner struct {
	BaseURL string
}

func NewPresigner(baseURL string) *Presigner {
	return &Presigner{BaseURL: baseURL}
}

func (p *Presigner) PresignPut(_ context.Context, objectKey string, _ string, _ time.Duration) (string, error) {
	return p.BaseURL + "/local-upload/" + url.PathEscape(objectKey), nil
}
