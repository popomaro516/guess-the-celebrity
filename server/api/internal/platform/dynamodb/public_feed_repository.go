package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
)

type PublicFeedRepository struct {
	client    *awsdynamodb.Client
	tableName string
}

func NewPublicFeedRepository(
	client *awsdynamodb.Client,
	tableName string,
) *PublicFeedRepository {
	return &PublicFeedRepository{client: client, tableName: tableName}
}

func (r *PublicFeedRepository) FindPublicQuizCandidates(
	ctx context.Context,
	limit int,
) ([]quiz.PublicQuiz, error) {
	if limit <= 0 {
		return []quiz.PublicQuiz{}, nil
	}
	out, err := r.client.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"feed_id": stringAttr(publicFeedID),
		},
	})
	if err != nil {
		return nil, err
	}

	quizzes := getPublicQuizzes(out.Item, "quizzes")
	if len(quizzes) > limit {
		quizzes = quizzes[:limit]
	}
	return quizzes, nil
}

func (r *PublicFeedRepository) Remove(ctx context.Context, quizID string) error {
	out, err := r.client.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"feed_id": stringAttr(publicFeedID),
		},
	})
	if err != nil {
		return err
	}
	if len(out.Item) == 0 {
		return nil
	}

	quizzes := getPublicQuizzes(out.Item, "quizzes")
	filtered := make([]quiz.PublicQuiz, 0, len(quizzes))
	for _, publicQuiz := range quizzes {
		if publicQuiz.ID != quizID {
			filtered = append(filtered, publicQuiz)
		}
	}
	if len(filtered) == len(quizzes) {
		return nil
	}

	_, err = r.client.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"feed_id":    stringAttr(publicFeedID),
			"updated_at": timeAttr(time.Now()),
			"quizzes":    publicQuizzesAttr(filtered),
		},
	})
	return err
}

func getPublicQuizzes(item map[string]types.AttributeValue, name string) []quiz.PublicQuiz {
	entries, ok := item[name].(*types.AttributeValueMemberL)
	if !ok {
		return []quiz.PublicQuiz{}
	}
	quizzes := make([]quiz.PublicQuiz, 0, len(entries.Value))
	for _, entry := range entries.Value {
		value, ok := entry.(*types.AttributeValueMemberM)
		if !ok {
			continue
		}
		quizzes = append(quizzes, quiz.PublicQuiz{
			ID:              getString(value.Value, "quiz_id"),
			Question:        getString(value.Value, "question"),
			Choices:         getChoices(value.Value, "choices"),
			Difficulty:      quiz.Difficulty(getString(value.Value, "difficulty")),
			CroppedImageKey: getString(value.Value, "cropped_image_key"),
		})
	}
	return quizzes
}

func publicQuizzesAttr(quizzes []quiz.PublicQuiz) types.AttributeValue {
	values := make([]types.AttributeValue, 0, len(quizzes))
	for _, q := range quizzes {
		values = append(values, &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
			"quiz_id":           stringAttr(q.ID),
			"question":          stringAttr(q.Question),
			"choices":           choicesAttr(q.Choices),
			"difficulty":        stringAttr(string(q.Difficulty)),
			"cropped_image_key": stringAttr(q.CroppedImageKey),
		}})
	}
	return &types.AttributeValueMemberL{Value: values}
}
