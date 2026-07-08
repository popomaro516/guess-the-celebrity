package app

import (
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
	router.Use(requestLogger(logger), panicRecovery(logger))

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
		if !bindJSON(c, &req) {
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
		c.JSON(http.StatusOK, gin.H{
			"image_id": img.ID,
			"status":   img.Status,
		})
	})

	router.POST("/quizzes", deps.AuthMiddleware, func(c *gin.Context) {
		principal, ok := auth.PrincipalFromContext(c)
		if !ok {
			logger.ErrorContext(c.Request.Context(), "authenticated principal missing",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"route", c.FullPath(),
				"request_id", requestID(c),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
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
		if !bindJSON(c, &req) {
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
		c.JSON(http.StatusOK, gin.H{
			"quiz_id":           out.ID,
			"question":          out.Question,
			"cropped_image_url": assetURL(deps.AssetBaseURL, out.CroppedImageKey),
			"choices":           out.Choices,
			"difficulty":        out.Difficulty,
		})
	})

	router.POST("/quizzes/:quiz_id/publish", deps.AuthMiddleware, func(c *gin.Context) {
		out, err := deps.QuizService.Publish(c.Request.Context(), c.Param("quiz_id"))
		if err != nil {
			respondError(c, logger, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"quiz_id": out.ID,
			"status":  out.Status,
		})
	})

	router.POST("/quizzes/:quiz_id/answer", func(c *gin.Context) {
		var req struct {
			Answer string `json:"answer" binding:"required"`
		}
		if !bindJSON(c, &req) {
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

		response := gin.H{"correct": out.Correct}
		if out.Correct {
			response["correct_answer"] = out.CorrectAnswer
			response["original_image_url"] = assetURL(deps.AssetBaseURL, out.OriginalImageKey)
		}
		c.JSON(http.StatusOK, response)
	})

	return router
}

func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, image.ErrImageNotFound), errors.Is(err, quiz.ErrQuizNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		logger.ErrorContext(c.Request.Context(), "request failed",
			"error", err,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"route", c.FullPath(),
			"status", http.StatusInternalServerError,
			"request_id", requestID(c),
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
		)
	}
}

func requestID(c *gin.Context) string {
	if value := c.GetHeader("X-Request-Id"); value != "" {
		return value
	}
	return c.GetHeader("X-Amzn-Trace-Id")
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
