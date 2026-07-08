package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/attempt"
)

type AttemptRepository struct {
	client    *awsdynamodb.Client
	tableName string
}

func NewAttemptRepository(client *awsdynamodb.Client, tableName string) *AttemptRepository {
	return &AttemptRepository{client: client, tableName: tableName}
}

func (r *AttemptRepository) Save(ctx context.Context, a attempt.Attempt) error {
	_, err := r.client.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"PK":         stringAttr(quizPK(a.QuizID)),
			"SK":         stringAttr(attemptSK(a.ID)),
			"type":       stringAttr(attemptType),
			"attempt_id": stringAttr(a.ID),
			"quiz_id":    stringAttr(a.QuizID),
			"user_id":    stringAttr(a.UserID),
			"answer":     stringAttr(a.Answer),
			"is_correct": boolAttr(a.IsCorrect),
			"created_at": timeAttr(a.CreatedAt),
		},
	})
	return err
}
