package app_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tomy/guess-the-celebrity/server/api/internal/app"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/attempt"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/upload"
	"github.com/tomy/guess-the-celebrity/server/api/internal/platform/localdb"
	"github.com/tomy/guess-the-celebrity/server/api/internal/platform/localpresign"
	"github.com/tomy/guess-the-celebrity/server/api/internal/platform/localqueue"
	"github.com/tomy/guess-the-celebrity/server/api/internal/platform/localstorage"
)

func TestUploadAndCreateQuizRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := testRouter()

	presignBody := postJSON(t, router, "/uploads/presign", map[string]any{
		"filename":     "cat.jpg",
		"content_type": "image/jpeg",
		"size":         2048,
	}, http.StatusOK)
	imageID := presignBody["image_id"].(string)
	if imageID != "img_1" {
		t.Fatalf("image_id = %q, want img_1", imageID)
	}

	completeBody := postJSON(t, router, "/images/"+imageID+"/complete", nil, http.StatusOK)
	if completeBody["status"] != "uploaded" {
		t.Fatalf("status = %q, want uploaded", completeBody["status"])
	}

	quizBody := postJSON(t, router, "/quizzes", map[string]any{
		"image_id":   imageID,
		"question":   "これは何の動物？",
		"answer":     "cat",
		"choices":    []string{"cat", "dog", "fox", "rabbit"},
		"difficulty": "normal",
		"crop": map[string]float64{
			"x":      0.24,
			"y":      0.18,
			"width":  0.32,
			"height": 0.28,
		},
	}, http.StatusCreated)
	if quizBody["quiz_id"] != "quiz_2" {
		t.Fatalf("quiz_id = %q, want quiz_2", quizBody["quiz_id"])
	}
	if quizBody["status"] != "processing" {
		t.Fatalf("status = %q, want processing", quizBody["status"])
	}
}

func TestHealthRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := testRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if response["status"] != "ok" {
		t.Fatalf("status = %q, want ok", response["status"])
	}
}

func postJSON(t *testing.T, handler http.Handler, path string, body any, wantStatus int) map[string]any {
	t.Helper()

	var requestBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
			t.Fatalf("json encode: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, path, &requestBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("POST %s status = %d, want %d, body = %s", path, rec.Code, wantStatus, rec.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	return response
}

func testRouter() http.Handler {
	store := localdb.NewStore()
	imageRepo := localdb.NewImageRepository(store)
	quizRepo := localdb.NewQuizRepository(store)
	attemptRepo := localdb.NewAttemptRepository(store)
	ids := &sequenceIDs{}
	clock := fixedClock{}
	queue := localqueue.NewCropJobQueue()
	presigner := localpresign.NewPresigner("http://localhost:8080")
	objects := localstorage.NewObjectStore()

	return app.NewRouter(app.Dependencies{
		UploadService:  upload.NewService(imageRepo, presigner, objects, ids, clock),
		QuizService:    quiz.NewService(quizRepo, quizRepo, imageRepo, queue, ids, clock),
		AttemptService: attempt.NewService(attemptRepo, quizRepo, imageRepo, ids, clock),
		BaseURL:        "http://localhost:8080",
		AssetBaseURL:   "http://localhost:8080",
	})
}

type sequenceIDs struct {
	n int
}

func (s *sequenceIDs) NewID(prefix string) string {
	s.n++
	return prefix + "_" + string(rune('0'+s.n))
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
}
