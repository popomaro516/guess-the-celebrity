package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

func (r *PublicFeedRepository) FindPublicQuizCandidateIDs(
	ctx context.Context,
	limit int,
) ([]string, error) {
	if limit <= 0 {
		return nil, nil
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

	quizIDs := getStrings(out.Item, "quiz_ids")
	if len(quizIDs) > limit {
		quizIDs = quizIDs[:limit]
	}
	return quizIDs, nil
}
