package attempt

import (
	"errors"
	"time"
)

var ErrQuizNotPublished = errors.New("quiz is not published")

type Attempt struct {
	ID        string
	QuizID    string
	UserID    string
	Answer    string
	IsCorrect bool
	CreatedAt time.Time
}
