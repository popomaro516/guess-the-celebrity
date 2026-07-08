package dynamodb

import (
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/quiz"
)

const (
	metadataSK   = "METADATA"
	imageType    = "IMAGE"
	quizType     = "QUIZ"
	attemptType  = "ATTEMPT"
	publicFeedPK = "QUIZ_FEED#PUBLIC"
	feedLimit    = int32(10)
)

func imagePK(id string) string {
	return "IMAGE#" + id
}

func quizPK(id string) string {
	return "QUIZ#" + id
}

func attemptSK(id string) string {
	return "ATTEMPT#" + id
}

func stringAttr(value string) types.AttributeValue {
	return &types.AttributeValueMemberS{Value: value}
}

func numberAttr(value string) types.AttributeValue {
	return &types.AttributeValueMemberN{Value: value}
}

func boolAttr(value bool) types.AttributeValue {
	return &types.AttributeValueMemberBOOL{Value: value}
}

func timeAttr(value time.Time) types.AttributeValue {
	return stringAttr(value.UTC().Format(time.RFC3339Nano))
}

func floatAttr(value float64) types.AttributeValue {
	return numberAttr(strconv.FormatFloat(value, 'f', -1, 64))
}

func int64Attr(value int64) types.AttributeValue {
	return numberAttr(strconv.FormatInt(value, 10))
}

func choicesAttr(choices []string) types.AttributeValue {
	values := make([]types.AttributeValue, 0, len(choices))
	for _, choice := range choices {
		values = append(values, stringAttr(choice))
	}
	return &types.AttributeValueMemberL{Value: values}
}

func getString(item map[string]types.AttributeValue, name string) string {
	value, ok := item[name].(*types.AttributeValueMemberS)
	if !ok {
		return ""
	}
	return value.Value
}

func getInt64(item map[string]types.AttributeValue, name string) int64 {
	value, ok := item[name].(*types.AttributeValueMemberN)
	if !ok {
		return 0
	}
	parsed, _ := strconv.ParseInt(value.Value, 10, 64)
	return parsed
}

func getFloat64(item map[string]types.AttributeValue, name string) float64 {
	value, ok := item[name].(*types.AttributeValueMemberN)
	if !ok {
		return 0
	}
	parsed, _ := strconv.ParseFloat(value.Value, 64)
	return parsed
}

func getTime(item map[string]types.AttributeValue, name string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, getString(item, name))
	return parsed
}

func getChoices(item map[string]types.AttributeValue, name string) []string {
	value, ok := item[name].(*types.AttributeValueMemberL)
	if !ok {
		return nil
	}
	choices := make([]string, 0, len(value.Value))
	for _, entry := range value.Value {
		choice, ok := entry.(*types.AttributeValueMemberS)
		if ok {
			choices = append(choices, choice.Value)
		}
	}
	return choices
}

func quizCrop(item map[string]types.AttributeValue) quiz.Crop {
	return quiz.Crop{
		X:      getFloat64(item, "crop_x_ratio"),
		Y:      getFloat64(item, "crop_y_ratio"),
		Width:  getFloat64(item, "crop_width_ratio"),
		Height: getFloat64(item, "crop_height_ratio"),
	}
}
