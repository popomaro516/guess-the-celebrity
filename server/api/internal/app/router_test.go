package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tomy/guess-the-celebrity/server/api/internal/app"
	"github.com/tomy/guess-the-celebrity/server/api/internal/auth"
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

func TestCreateQuizStoresAuthenticatedUserSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, quizRepo := newTestRouter(auth.Require(acceptTokens{}))

	presignBody := postJSON(t, router, "/uploads/presign", map[string]any{
		"filename":     "cat.jpg",
		"content_type": "image/jpeg",
		"size":         2048,
	}, http.StatusOK)
	imageID := presignBody["image_id"].(string)
	postJSON(t, router, "/images/"+imageID+"/complete", nil, http.StatusOK)

	quizBody := postJSON(t, router, "/quizzes", map[string]any{
		"image_id":   imageID,
		"question":   "これは何の動物？",
		"answer":     "cat",
		"choices":    []string{"cat", "dog", "fox", "rabbit"},
		"difficulty": "normal",
		"crop": map[string]float64{
			"x": 0.24, "y": 0.18, "width": 0.32, "height": 0.28,
		},
	}, http.StatusCreated)

	saved, err := quizRepo.FindByID(context.Background(), quizBody["quiz_id"].(string))
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if saved.CreatorUserID != "cognito-sub-123" {
		t.Fatalf("creator user ID = %q, want cognito-sub-123", saved.CreatorUserID)
	}
}

func TestAnswerRouteRevealsAnswerAndOriginalImageWhenWrong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, quizRepo := newTestRouter(auth.Disabled())

	presignBody := postJSON(t, router, "/uploads/presign", map[string]any{
		"filename":     "cat.jpg",
		"content_type": "image/jpeg",
		"size":         2048,
	}, http.StatusOK)
	imageID := presignBody["image_id"].(string)
	postJSON(t, router, "/images/"+imageID+"/complete", nil, http.StatusOK)

	quizBody := postJSON(t, router, "/quizzes", map[string]any{
		"image_id":   imageID,
		"question":   "これは何の動物？",
		"answer":     "cat",
		"choices":    []string{"cat", "dog", "fox", "rabbit"},
		"difficulty": "normal",
		"crop": map[string]float64{
			"x": 0.24, "y": 0.18, "width": 0.32, "height": 0.28,
		},
	}, http.StatusCreated)
	quizID := quizBody["quiz_id"].(string)
	saved, err := quizRepo.FindByID(context.Background(), quizID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	saved.Status = quiz.StatusPublished
	if err := quizRepo.Save(context.Background(), saved); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	answerBody := postJSON(t, router, "/quizzes/"+quizID+"/answer", map[string]any{
		"answer": "dog",
	}, http.StatusOK)
	if answerBody["correct"] != false {
		t.Fatalf("correct = %v, want false", answerBody["correct"])
	}
	if answerBody["correct_answer"] != "cat" {
		t.Fatalf("correct_answer = %q, want cat", answerBody["correct_answer"])
	}
	if answerBody["original_image_url"] != "http://localhost:8080/originals/anonymous/img_1/source.jpg" {
		t.Fatalf("original_image_url = %q", answerBody["original_image_url"])
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

func TestAuthoringRoutesRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := app.NewRouter(app.Dependencies{
		AuthMiddleware: auth.Require(rejectTokens{}),
	})

	paths := []string{
		"/uploads/presign",
		"/images/img_1/complete",
		"/quizzes",
		"/quizzes/quiz_1/publish",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("POST %s status = %d, want %d", path, rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestDeleteQuizRouteRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := app.NewRouter(app.Dependencies{
		AuthMiddleware: auth.Require(rejectTokens{}),
	})

	req := httptest.NewRequest(http.MethodDelete, "/quizzes/quiz_1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("DELETE /quizzes/quiz_1 status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMyQuizzesRouteRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := app.NewRouter(app.Dependencies{
		AuthMiddleware: auth.Require(rejectTokens{}),
	})

	req := httptest.NewRequest(http.MethodGet, "/quizzes/mine", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /quizzes/mine status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMyQuizzesRouteReturnsAuthenticatedUsersQuizzes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, quizRepo := newTestRouter(auth.Require(acceptTokens{}))
	createdAt := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	for _, q := range []quiz.Quiz{
		{
			ID:              "quiz_owned",
			CreatorUserID:   "cognito-sub-123",
			Question:        "これは誰？",
			Difficulty:      quiz.DifficultyNormal,
			Status:          quiz.StatusReady,
			CroppedImageKey: "quizzes/quiz_owned/crop.webp",
			CreatedAt:       createdAt,
		},
		{
			ID:            "quiz_other",
			CreatorUserID: "other-user",
			Question:      "他人の問題",
			Difficulty:    quiz.DifficultyHard,
			Status:        quiz.StatusPublished,
			CreatedAt:     createdAt.Add(time.Hour),
		},
	} {
		if err := quizRepo.Save(context.Background(), q); err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/quizzes/mine", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /quizzes/mine status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response struct {
		Quizzes []struct {
			ID              string          `json:"quiz_id"`
			Question        string          `json:"question"`
			Difficulty      quiz.Difficulty `json:"difficulty"`
			Status          quiz.Status     `json:"status"`
			CroppedImageURL string          `json:"cropped_image_url"`
			CreatedAt       string          `json:"created_at"`
		} `json:"quizzes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if len(response.Quizzes) != 1 {
		t.Fatalf("len(quizzes) = %d, want 1", len(response.Quizzes))
	}
	got := response.Quizzes[0]
	if got.ID != "quiz_owned" || got.Question != "これは誰？" {
		t.Fatalf("unexpected quiz: %+v", got)
	}
	if got.Difficulty != quiz.DifficultyNormal || got.Status != quiz.StatusReady {
		t.Fatalf("unexpected quiz metadata: %+v", got)
	}
	if got.CroppedImageURL != "http://localhost:8080/quizzes/quiz_owned/crop.webp" {
		t.Fatalf("cropped_image_url = %q", got.CroppedImageURL)
	}
	if got.CreatedAt != createdAt.Format(time.RFC3339Nano) {
		t.Fatalf("created_at = %q, want %q", got.CreatedAt, createdAt.Format(time.RFC3339Nano))
	}
}

func TestMyQuizzesRouteReturnsEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := newTestRouter(auth.Require(acceptTokens{}))

	req := httptest.NewRequest(http.MethodGet, "/quizzes/mine", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /quizzes/mine status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != `{"quizzes":[]}` {
		t.Fatalf("body = %s, want empty quizzes array", rec.Body.String())
	}
}

func TestMyQuizzesRouteOmitsCropURLWhileProcessing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, quizRepo := newTestRouter(auth.Require(acceptTokens{}))
	if err := quizRepo.Save(context.Background(), quiz.Quiz{
		ID:              "quiz_processing",
		CreatorUserID:   "cognito-sub-123",
		Status:          quiz.StatusProcessing,
		CroppedImageKey: "quizzes/quiz_processing/crop.webp",
	}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/quizzes/mine", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var response struct {
		Quizzes []map[string]any `json:"quizzes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if _, exists := response.Quizzes[0]["cropped_image_url"]; exists {
		t.Fatalf("processing quiz must not include cropped_image_url: %s", rec.Body.String())
	}
}

func TestPublishRejectsUserWhoIsNotCreator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, quizRepo := newTestRouter(auth.Require(acceptTokens{}))
	if err := quizRepo.Save(context.Background(), quiz.Quiz{
		ID:            "quiz_123",
		CreatorUserID: "owner-123",
		Status:        quiz.StatusReady,
	}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	body := postJSON(t, router, "/quizzes/quiz_123/publish", nil, http.StatusForbidden)
	if body["error"] != "forbidden" {
		t.Fatalf("error = %q, want forbidden", body["error"])
	}
}

func TestDeleteQuizRouteRemovesAuthenticatedUsersQuiz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, quizRepo := newTestRouter(auth.Require(acceptTokens{}))

	presignBody := postJSON(t, router, "/uploads/presign", map[string]any{
		"filename":     "cat.jpg",
		"content_type": "image/jpeg",
		"size":         2048,
	}, http.StatusOK)
	imageID := presignBody["image_id"].(string)
	postJSON(t, router, "/images/"+imageID+"/complete", nil, http.StatusOK)

	quizBody := postJSON(t, router, "/quizzes", map[string]any{
		"image_id":   imageID,
		"question":   "これは何の動物？",
		"answer":     "cat",
		"choices":    []string{"cat", "dog", "fox", "rabbit"},
		"difficulty": "normal",
		"crop": map[string]float64{
			"x": 0.24, "y": 0.18, "width": 0.32, "height": 0.28,
		},
	}, http.StatusCreated)
	quizID := quizBody["quiz_id"].(string)

	deleteNoContent(t, router, "/quizzes/"+quizID, http.StatusNoContent)
	if _, err := quizRepo.FindByID(context.Background(), quizID); !errors.Is(err, quiz.ErrQuizNotFound) {
		t.Fatalf("FindByID err = %v, want %v", err, quiz.ErrQuizNotFound)
	}
}

func TestDeleteQuizRouteRejectsUserWhoIsNotCreator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, quizRepo := newTestRouter(auth.Require(acceptTokens{}))
	if err := quizRepo.Save(context.Background(), quiz.Quiz{
		ID:              "quiz_123",
		CreatorUserID:   "owner-123",
		ImageID:         "img_123",
		CroppedImageKey: "quizzes/quiz_123/crop.webp",
	}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	body := deleteJSON(t, router, "/quizzes/quiz_123", http.StatusForbidden)
	if body["error"] != "forbidden" {
		t.Fatalf("error = %q, want forbidden", body["error"])
	}
	if _, err := quizRepo.FindByID(context.Background(), "quiz_123"); err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
}

func TestRouterAssignsRequestIDAndLogsPrincipalSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
	router, _ := newTestRouterWithLogger(auth.Require(acceptTokens{}), logger)

	req := httptest.NewRequest(http.MethodGet, "/quizzes/mine", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /quizzes/mine status = %d, want %d", rec.Code, http.StatusOK)
	}
	requestID := rec.Header().Get("X-Request-Id")
	if requestID == "" {
		t.Fatal("X-Request-Id header is empty")
	}
	logs := logOutput.String()
	if !strings.Contains(logs, `"msg":"http request"`) {
		t.Fatalf("request log missing: %s", logs)
	}
	if !strings.Contains(logs, `"principal_subject":"cognito-sub-123"`) {
		t.Fatalf("principal_subject missing from logs: %s", logs)
	}
	if !strings.Contains(logs, `"request_id":"`+requestID+`"`) {
		t.Fatalf("request_id %q missing from logs: %s", requestID, logs)
	}
}

func TestCreateQuizLogsStructuredEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
	router, _ := newTestRouterWithLogger(auth.Disabled(), logger)

	presignBody := postJSON(t, router, "/uploads/presign", map[string]any{
		"filename":     "cat.jpg",
		"content_type": "image/jpeg",
		"size":         2048,
	}, http.StatusOK)
	imageID := presignBody["image_id"].(string)
	postJSON(t, router, "/images/"+imageID+"/complete", nil, http.StatusOK)

	postJSON(t, router, "/quizzes", map[string]any{
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

	logs := logOutput.String()
	if !strings.Contains(logs, `"msg":"quiz created"`) {
		t.Fatalf("quiz created log missing: %s", logs)
	}
	if !strings.Contains(logs, `"quiz_id":"quiz_2"`) {
		t.Fatalf("quiz_id missing from logs: %s", logs)
	}
	if !strings.Contains(logs, `"creator_user_id":"local-development-user"`) {
		t.Fatalf("creator_user_id missing from logs: %s", logs)
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
	req.Header.Set("Authorization", "Bearer valid-token")
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

func deleteNoContent(t *testing.T, handler http.Handler, path string, wantStatus int) {
	t.Helper()

	req := httptest.NewRequest(http.MethodDelete, path, nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("DELETE %s status = %d, want %d, body = %s", path, rec.Code, wantStatus, rec.Body.String())
	}
}

func deleteJSON(t *testing.T, handler http.Handler, path string, wantStatus int) map[string]any {
	t.Helper()

	req := httptest.NewRequest(http.MethodDelete, path, nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("DELETE %s status = %d, want %d, body = %s", path, rec.Code, wantStatus, rec.Body.String())
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	return response
}

func testRouter() http.Handler {
	router, _ := newTestRouter(auth.Disabled())
	return router
}

func newTestRouter(authMiddleware gin.HandlerFunc) (http.Handler, *localdb.QuizRepository) {
	return newTestRouterWithLogger(authMiddleware, nil)
}

func newTestRouterWithLogger(authMiddleware gin.HandlerFunc, logger *slog.Logger) (http.Handler, *localdb.QuizRepository) {
	store := localdb.NewStore()
	imageRepo := localdb.NewImageRepository(store)
	quizRepo := localdb.NewQuizRepository(store)
	ids := &sequenceIDs{}
	clock := fixedClock{}
	queue := localqueue.NewCropJobQueue()
	presigner := localpresign.NewPresigner("http://localhost:8080")
	objects := localstorage.NewObjectStore()

	router := app.NewRouter(app.Dependencies{
		UploadService:  upload.NewService(imageRepo, presigner, objects, ids, clock),
		QuizService:    quiz.NewService(quizRepo, quizRepo, imageRepo, objects, queue, ids, clock),
		AttemptService: attempt.NewService(quizRepo, imageRepo),
		AuthMiddleware: authMiddleware,
		BaseURL:        "http://localhost:8080",
		AssetBaseURL:   "http://localhost:8080",
		Logger:         logger,
	})
	return router, quizRepo
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

type rejectTokens struct{}

func (rejectTokens) Verify(context.Context, string) (auth.Principal, error) {
	return auth.Principal{}, errors.New("invalid token")
}

type acceptTokens struct{}

func (acceptTokens) Verify(_ context.Context, token string) (auth.Principal, error) {
	if token != "valid-token" {
		return auth.Principal{}, errors.New("invalid token")
	}
	return auth.Principal{Subject: "cognito-sub-123"}, nil
}
