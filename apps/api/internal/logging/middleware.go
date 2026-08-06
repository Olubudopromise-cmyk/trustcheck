package logging

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// requestIDKey is the context key under which the request ID is stored.
type requestIDKey struct{}

// responseRecorder wraps the gin response writer so the logger can observe
// the response status, capture any error body, and collect extra attributes
// added by handlers. It emits a single structured entry per request.
type responseRecorder struct {
	gin.ResponseWriter
	status     int
	captureErr bool
	errBody    bytes.Buffer
	errMessage string
	attrs      []any
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	if code >= http.StatusBadRequest {
		r.captureErr = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.captureErr {
		r.errBody.Write(b)
	}
	return r.ResponseWriter.Write(b)
}

// errorMessage returns the failure description for a failed request: a panic
// message set by the recovery middleware, the "error" field of a JSON error
// response, or the raw response body as a fallback.
func (r *responseRecorder) errorMessage() string {
	if r.errMessage != "" {
		return r.errMessage
	}
	body := strings.TrimSpace(r.errBody.String())
	if body == "" {
		return ""
	}
	var parsed struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err == nil && parsed.Error != "" {
		return parsed.Error
	}
	return body
}

// RequestID injects a request ID into the request context and the
// X-Request-ID response header. The client's own X-Request-ID value is
// reused when present; otherwise a cryptographically random hex ID is
// generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if id == "" {
			id = newRequestID()
		}
		c.Header("X-Request-ID", id)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDKey{}, id))
		c.Next()
	}
}

// GetRequestID returns the request ID stored in the context, if any.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// AddRequestAttrs attaches extra key/value pairs to the log entry of the
// current request. It must be called from a handler running under
// RequestLogger.
func AddRequestAttrs(c *gin.Context, attrs ...any) {
	rec, ok := c.Writer.(*responseRecorder)
	if !ok {
		return
	}
	rec.attrs = append(rec.attrs, attrs...)
}

// RequestLogger records the start time, runs the handler, then emits exactly
// one structured JSON log entry per request. Requests that complete with an
// HTTP status of 400 or higher are logged at error level with an "error"
// field.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = rec
		c.Next()

		args := append([]any{
			"requestId", GetRequestID(c.Request.Context()),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", rec.status,
			"latencyMs", time.Since(start).Milliseconds(),
			"clientIP", c.ClientIP(),
			"userAgent", c.Request.UserAgent(),
		}, rec.attrs...)

		if rec.status >= http.StatusBadRequest {
			args = append(args, "error", rec.errorMessage())
			logger.Error("request failed", args...)
			return
		}
		logger.Info("request completed", args...)
	}
}

// Recovery converts a panicking handler into a 500 response and records the
// panic message as the request error, so the request still produces exactly
// one structured log entry.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if rec, ok := c.Writer.(*responseRecorder); ok {
					rec.errMessage = fmt.Sprintf("panic: %v", err)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

// newRequestID returns a 128-bit cryptographically random ID as hex.
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
