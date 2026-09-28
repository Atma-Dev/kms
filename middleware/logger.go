package middleware

import (
	"bytes"
	"context"
	"io"
	"api-kms/app/activitylog"
	"api-kms/app/user"
	"api-kms/config"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const loggerKey = "logger"

// bodyCaptureWriter tees the response body into an in-memory buffer (capped
// at activitylog.MaxBodyCaptureBytes) alongside the real write, so
// LoggerMiddleware can persist it without buffering an unbounded response
// (a CSV export, a big list payload) in memory.
type bodyCaptureWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	if remaining := activitylog.MaxBodyCaptureBytes - w.buf.Len(); remaining > 0 {
		if remaining > len(b) {
			remaining = len(b)
		}
		w.buf.Write(b[:remaining])
	}
	return w.ResponseWriter.Write(b)
}

// WriteString is part of gin.ResponseWriter and bypasses Write above, so it
// needs its own capture path — otherwise plain-text/string responses would
// go unlogged.
func (w *bodyCaptureWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func LoggerMiddleware(hub *activitylog.Hub) gin.HandlerFunc {
	broadcaster := activitylog.NewBroadcaster(activitylog.NewRepository(config.DB), hub)

	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("RequestID", requestID)

		entry := config.AppLogger.WithFields(logrus.Fields{
			"request_id": requestID,
			"ip":         c.ClientIP(),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
		})
		c.Set(loggerKey, entry)

		shouldLog := activitylog.ShouldLog(c.Request.Method, c.Request.URL.Path)

		var requestBody []byte
		var writer *bodyCaptureWriter
		if shouldLog {
			requestBody = captureRequestBody(c)
			writer = &bodyCaptureWriter{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}
			c.Writer = writer
		}

		start := time.Now()
		c.Next()

		if c.Request.Method == http.MethodOptions {
			return
		}

		entry.WithFields(logrus.Fields{
			"status":  c.Writer.Status(),
			"latency": time.Since(start).String(),
		}).Info("request completed")

		if shouldLog {
			recordActivity(broadcaster, c, entry, requestBody, writer.buf.Bytes())
		}
	}
}

// captureRequestBody reads and restores the request body so the handler
// still sees it whole. Multipart bodies (file uploads) are left untouched —
// buffering them into memory for a log row isn't worth it, and they aren't
// useful to read back as text anyway.
func captureRequestBody(c *gin.Context) []byte {
	if c.Request.Body == nil || strings.HasPrefix(c.Request.Header.Get("Content-Type"), "multipart/") {
		return nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

// recordActivity persists the request as an activity log row and pushes it
// to any connected live-feed viewers. Fired in a goroutine on its own
// context since the request's context is canceled once the response is
// written, right before this runs.
func recordActivity(broadcaster *activitylog.Broadcaster, c *gin.Context, entry *logrus.Entry, requestBody, responseBody []byte) {
	log := &activitylog.ActivityLog{
		Method:       c.Request.Method,
		Path:         c.Request.URL.Path,
		StatusCode:   c.Writer.Status(),
		IP:           c.ClientIP(),
		RequestBody:  activitylog.SanitizeBody(requestBody, c.Request.Header.Get("Content-Type")),
		ResponseBody: activitylog.SanitizeBody(responseBody, c.Writer.Header().Get("Content-Type")),
	}
	if authUser, ok := c.Get("user"); ok {
		if u, ok := authUser.(*user.User); ok {
			id := u.ID.String()
			log.UserID = &id
			// Already have the full user from context — reuse it instead of
			// a Preload("User") query, since NewResource(log) reads it for
			// the broadcast payload.
			log.User = u
		}
	}

	go func() {
		if err := broadcaster.Record(context.Background(), log); err != nil {
			entry.WithError(err).Error("failed to write activity log")
		}
	}()
}

func Logger(c *gin.Context) *logrus.Entry {
	if entry, ok := c.Get(loggerKey); ok {
		return entry.(*logrus.Entry)
	}
	return config.AppLogger.WithField("ip", c.ClientIP())
}
