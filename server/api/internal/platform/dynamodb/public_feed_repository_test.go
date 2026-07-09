package dynamodb

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
)

func TestGetPublicQuizzesDecodesFeedProjection(t *testing.T) {
	item := map[string]types.AttributeValue{
		"quizzes": &types.AttributeValueMemberL{Value: []types.AttributeValue{
			&types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"quiz_id":           stringAttr("quiz_123"),
				"question":          stringAttr("この人物は誰？"),
				"choices":           choicesAttr([]string{"A", "B", "C", "D"}),
				"difficulty":        stringAttr("hard"),
				"cropped_image_key": stringAttr("quizzes/quiz_123/crop.webp"),
			}},
		}},
	}

	got := getPublicQuizzes(item, "quizzes")

	if len(got) != 1 {
		t.Fatalf("len(quizzes) = %d, want 1", len(got))
	}
	want := quiz.PublicQuiz{
		ID:              "quiz_123",
		Question:        "この人物は誰？",
		Choices:         []string{"A", "B", "C", "D"},
		Difficulty:      quiz.DifficultyHard,
		CroppedImageKey: "quizzes/quiz_123/crop.webp",
	}
	if got[0].ID != want.ID ||
		got[0].Question != want.Question ||
		got[0].Difficulty != want.Difficulty ||
		got[0].CroppedImageKey != want.CroppedImageKey ||
		len(got[0].Choices) != len(want.Choices) {
		t.Fatalf("quiz = %+v, want %+v", got[0], want)
	}
}

func TestGetPublicQuizzesReturnsEmptySliceForMissingFeed(t *testing.T) {
	got := getPublicQuizzes(nil, "quizzes")

	if got == nil {
		t.Fatal("quizzes = nil, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("len(quizzes) = %d, want 0", len(got))
	}
}
