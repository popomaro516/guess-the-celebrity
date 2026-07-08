package sqs

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/job"
)

type CropJobQueue struct {
	client   *awssqs.Client
	queueURL string
}

func NewCropJobQueue(client *awssqs.Client, queueURL string) *CropJobQueue {
	return &CropJobQueue{client: client, queueURL: queueURL}
}

func (q *CropJobQueue) EnqueueCropJob(ctx context.Context, cropJob job.CropJob) error {
	body, err := json.Marshal(cropJob)
	if err != nil {
		return err
	}
	_, err = q.client.SendMessage(ctx, &awssqs.SendMessageInput{
		QueueUrl:    aws.String(q.queueURL),
		MessageBody: aws.String(string(body)),
	})
	return err
}
