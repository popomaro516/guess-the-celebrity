package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tomy/guess-the-celebrity/server/api/internal/auth"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/attempt"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/image"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/upload"
)

type Dependencies struct {
	UploadService  *upload.Service
	QuizService    *quiz.Service
	AttemptService *attempt.Service
	AuthMiddleware gin.HandlerFunc
	BaseURL        string
	AssetBaseURL   string
	Logger         *slog.Logger
}

func NewRouter(deps Dependencies) *gin.Engine {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	router := gin.New()
	router.Use(requestMetadata(), requestLogger(logger), panicRecovery(logger))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.POST("/uploads/presign", deps.AuthMiddleware, func(c *gin.Context) {
		var req struct {
			Filename    string `json:"filename" binding:"required"`
			ContentType string `json:"content_type" binding:"required"`
			Size        int64  `json:"size" binding:"required"`
		}
		if !bindJSON(c, logger, &req) {
			return
		}
		out, err := deps.UploadService.Presign(c.Request.Context(), upload.PresignInput{
			Filename:    req.Filename,
			ContentType: req.ContentType,
			Size:        req.Size,
		})
		if err != nil {
			respondError(c, logger, err)
			return
		}
		logEvent(c, logger, slog.LevelInfo, "upload presigned",
			slog.String("image_id", out.ImageID),
			slog.String("object_key", out.ObjectKey),
			slog.String("content_type", req.ContentType),
			slog.Int64("size_bytes", req.Size),
			slog.Int("expires_in_seconds", out.ExpiresIn),
		)
		c.JSON(http.StatusOK, gin.H{
			"image_id":   out.ImageID,
			"upload_url": out.UploadURL,
			"object_key": out.ObjectKey,
			"expires_in": out.ExpiresIn,
		})
	})

	router.POST("/images/:image_id/complete", deps.AuthMiddleware, func(c *gin.Context) {
		img, err := deps.UploadService.Complete(c.Request.Context(), c.Param("image_id"))
		if err != nil {
			respondError(c, logger, err)
			return
		}
		logEvent(c, logger, slog.LevelInfo, "upload completed",
			slog.String("image_id", img.ID),
			slog.String("status", string(img.Status)),
		)
		c.JSON(http.StatusOK, gin.H{
			"image_id": img.ID,
			"status":   img.Status,
		})
	})

	router.POST("/quizzes", deps.AuthMiddleware, func(c *gin.Context) {
		principal, ok := authenticatedPrincipal(c, logger)
		if !ok {
			return
		}
		var req struct {
			ImageID    string          `json:"image_id" binding:"required"`
			Question   string          `json:"question" binding:"required"`
			Answer     string          `json:"answer" binding:"required"`
			Choices    []string        `json:"choices" binding:"required"`
			Difficulty quiz.Difficulty `json:"difficulty" binding:"required"`
			Crop       quiz.Crop       `json:"crop" binding:"required"`
		}
		if !bindJSON(c, logger, &req) {
			return
		}
		out, err := deps.QuizService.Create(c.Request.Context(), principal.Subject, quiz.CreateInput{
			ImageID:    req.ImageID,
			Question:   req.Question,
			Answer:     req.Answer,
			Choices:    req.Choices,
			Difficulty: req.Difficulty,
			Crop:       req.Crop,
		})
		if err != nil {
			respondError(c, logger, err)
			return
		}
		logEvent(c, logger, slog.LevelInfo, "quiz created",
			slog.String("quiz_id", out.ID),
			slog.String("image_id", req.ImageID),
			slog.String("difficulty", string(req.Difficulty)),
			slog.Int("choices_count", len(req.Choices)),
			slog.String("creator_user_id", principal.Subject),
			slog.String("status", string(out.Status)),
		)
		c.JSON(http.StatusCreated, gin.H{
			"quiz_id": out.ID,
			"status":  out.Status,
		})
	})

	router.GET("/quizzes/random", func(c *gin.Context) {
		out, err := deps.QuizService.RandomPublished(c.Request.Context())
		if err != nil {
			respondError(c, logger, err)
			return
		}
		logEvent(c, logger, slog.LevelInfo, "random quiz served",
			slog.String("quiz_id", out.ID),
			slog.String("difficulty", string(out.Difficulty)),
		)
		c.JSON(http.StatusOK, gin.H{
			"quiz_id":           out.ID,
			"question":          out.Question,
			"cropped_image_url": assetURL(deps.AssetBaseURL, out.CroppedImageKey),
			"choices":           out.Choices,
			"difficulty":        out.Difficulty,
		})
	})

	router.GET("/quizzes/mine", deps.AuthMiddleware, func(c *gin.Context) {
		principal, ok := authenticatedPrincipal(c, logger)
		if !ok {
			return
		}
		quizzes, err := deps.QuizService.ListOwned(c.Request.Context(), principal.Subject)
		if err != nil {
			respondError(c, logger, err)
			return
		}
		logEvent(c, logger, slog.LevelInfo, "owned quizzes listed",
			slog.Int("quiz_count", len(quizzes)),
		)
		response := make([]gin.H, 0, len(quizzes))
		for _, ownedQuiz := range quizzes {
			item := gin.H{
				"quiz_id":    ownedQuiz.ID,
				"question":   ownedQuiz.Question,
				"difficulty": ownedQuiz.Difficulty,
				"status":     ownedQuiz.Status,
				"created_at": ownedQuiz.CreatedAt,
			}
			if ownedQuiz.Status == quiz.StatusReady || ownedQuiz.Status == quiz.StatusPublished {
				item["cropped_image_url"] = assetURL(deps.AssetBaseURL, ownedQuiz.CroppedImageKey)
			}
			response = append(response, item)
		}
		c.JSON(http.StatusOK, gin.H{"quizzes": response})
	})

	router.POST("/quizzes/:quiz_id/publish", deps.AuthMiddleware, func(c *gin.Context) {
		principal, ok := authenticatedPrincipal(c, logger)
		if !ok {
			return
		}
		out, err := deps.QuizService.Publish(c.Request.Context(), principal.Subject, c.Param("quiz_id"))
		if err != nil {
			respondError(c, logger, err)
			return
		}
		logEvent(c, logger, slog.LevelInfo, "quiz published",
			slog.String("quiz_id", out.ID),
			slog.String("status", string(out.Status)),
		)
		c.JSON(http.StatusOK, gin.H{
			"quiz_id": out.ID,
			"status":  out.Status,
		})
	})

	router.DELETE("/quizzes/:quiz_id", deps.AuthMiddleware, func(c *gin.Context) {
		principal, ok := authenticatedPrincipal(c, logger)
		if !ok {
			return
		}
		if err := deps.QuizService.Delete(c.Request.Context(), principal.Subject, c.Param("quiz_id")); err != nil {
			respondError(c, logger, err)
			return
		}
		logEvent(c, logger, slog.LevelInfo, "quiz deleted",
			slog.String("quiz_id", c.Param("quiz_id")),
			slog.String("creator_user_id", principal.Subject),
		)
		c.Status(http.StatusNoContent)
	})

	router.POST("/quizzes/:quiz_id/answer", func(c *gin.Context) {
		var req struct {
			Answer string `json:"answer" binding:"required"`
		}
		if !bindJSON(c, logger, &req) {
			return
		}
		out, err := deps.AttemptService.Answer(c.Request.Context(), attempt.AnswerInput{
			QuizID: c.Param("quiz_id"),
			Answer: req.Answer,
		})
		if err != nil {
			respondError(c, logger, err)
			return
		}
		logEvent(c, logger, slog.LevelInfo, "quiz answered",
			slog.String("quiz_id", c.Param("quiz_id")),
			slog.Bool("correct", out.Correct),
		)

		response := gin.H{"correct": out.Correct}
		if out.CorrectAnswer != "" {
			response["correct_answer"] = out.CorrectAnswer
		}
		if out.OriginalImageKey != "" {
			response["original_image_url"] = assetURL(deps.AssetBaseURL, out.OriginalImageKey)
		}
		c.JSON(http.StatusOK, response)
	})

	return router
}

func authenticatedPrincipal(c *gin.Context, logger *slog.Logger) (auth.Principal, bool) {
	principal, ok := auth.PrincipalFromContext(c)
	if ok {
		return principal, true
	}
	logger.ErrorContext(c.Request.Context(), "authenticated principal missing",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"route", c.FullPath(),
		"request_id", requestID(c),
	)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	return auth.Principal{}, false
}

func bindJSON(c *gin.Context, logger *slog.Logger, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		logEvent(c, logger, slog.LevelWarn, "request validation failed",
			slog.Int("status", http.StatusBadRequest),
			slog.Any("error", err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return false
	}
	return true
}

func respondError(c *gin.Context, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, upload.ErrInvalidUpload),
		errors.Is(err, quiz.ErrInvalidCrop),
		errors.Is(err, quiz.ErrInvalidChoices),
		errors.Is(err, quiz.ErrImageNotUploaded),
		errors.Is(err, quiz.ErrQuizNotReady),
		errors.Is(err, attempt.ErrQuizNotPublished),
		errors.Is(err, upload.ErrUploadObjectNotFound):
		logEvent(c, logger, slog.LevelWarn, "request failed",
			slog.Int("status", http.StatusBadRequest),
			slog.Any("error", err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, image.ErrImageNotFound), errors.Is(err, quiz.ErrQuizNotFound):
		logEvent(c, logger, slog.LevelWarn, "request failed",
			slog.Int("status", http.StatusNotFound),
			slog.Any("error", err),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, quiz.ErrPublishForbidden), errors.Is(err, quiz.ErrDeleteForbidden):
		logEvent(c, logger, slog.LevelWarn, "request failed",
			slog.Int("status", http.StatusForbidden),
			slog.Any("error", err),
		)
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	default:
		logEvent(c, logger, slog.LevelError, "request failed",
			slog.Int("status", http.StatusInternalServerError),
			slog.Any("error", err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func assetURL(baseURL, key string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(key, "/")
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		if status >= http.StatusInternalServerError {
			level = slog.LevelError
		} else if status >= http.StatusBadRequest {
			level = slog.LevelWarn
		}

		logger.LogAttrs(c.Request.Context(), level, "http request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("route", c.FullPath()),
			slog.Int("status", status),
			slog.Int("bytes", c.Writer.Size()),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("request_id", requestID(c)),
			slog.String("principal_subject", principalSubject(c)),
			slog.String("user_agent", c.Request.UserAgent()),
		)
	}
}

const requestIDKey = "request.id"

func requestMetadata() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := requestID(c)
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-Id", requestID)
		c.Next()
	}
}

func requestID(c *gin.Context) string {
	if value, ok := c.Get(requestIDKey); ok {
		if requestID, ok := value.(string); ok && requestID != "" {
			return requestID
		}
	}
	if value := c.GetHeader("X-Request-Id"); value != "" {
		return value
	}
	return c.GetHeader("X-Amzn-Trace-Id")
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return time.Now().UTC().Format("20060102T150405.000000000")
	}
	return hex.EncodeToString(bytes[:])
}

func principalSubject(c *gin.Context) string {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok || principal.Subject == "" {
		return "anonymous"
	}
	return principal.Subject
}

func logEvent(c *gin.Context, logger *slog.Logger, level slog.Level, msg string, attrs ...slog.Attr) {
	baseAttrs := []slog.Attr{
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.String("route", c.FullPath()),
		slog.String("request_id", requestID(c)),
		slog.String("principal_subject", principalSubject(c)),
	}
	logger.LogAttrs(c.Request.Context(), level, msg, append(baseAttrs, attrs...)...)
}

func panicRecovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		logger.ErrorContext(c.Request.Context(), "request panic",
			"panic", recovered,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"route", c.FullPath(),
			"status", http.StatusInternalServerError,
			"request_id", requestID(c),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	})
}
