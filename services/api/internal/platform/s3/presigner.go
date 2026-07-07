package s3

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Presigner struct {
	client *s3.PresignClient
	bucket string
}

func NewPresigner(client *s3.Client, bucket string) *Presigner {
	return &Presigner{client: s3.NewPresignClient(client), bucket: bucket}
}

func (p *Presigner) PresignPut(ctx context.Context, objectKey string, contentType string, expiresIn time.Duration) (string, error) {
	out, err := p.client.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, func(options *s3.PresignOptions) {
		options.Expires = expiresIn
	})
	if err != nil {
		return "", err
	}
	return out.URL, nil
}
