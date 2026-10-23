package internal

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newRatelimitTestRouter(t *testing.T, rate float64) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	app := &Application{RateLimiter: setupRatelimiting(Config{RateLimit: rate})}
	t.Cleanup(app.Shutdown)

	router := gin.New()
	router.Use(app.ratelimitMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	return router
}

func doRatelimitRequest(router *gin.Engine, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip + ":1234"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func TestRatelimitBlocksAfterLimit(t *testing.T) {
	router := newRatelimitTestRouter(t, 3)

	for i := range 3 {
		w := doRatelimitRequest(router, "192.0.2.1")
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, w.Code)
		}

		if got, want := w.Header().Get("X-RateLimit-Remaining"), strconv.Itoa(2-i); got != want {
			t.Errorf("request %d: expected remaining %s, got %s", i+1, want, got)
		}
	}

	w := doRatelimitRequest(router, "192.0.2.1")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after limit, got %d", w.Code)
	}

	retryAfter, err := strconv.Atoi(w.Header().Get("Retry-After"))
	if err != nil || retryAfter != 1 {
		t.Errorf("expected Retry-After of 1 second, got %q", w.Header().Get("Retry-After"))
	}

	reset, err := time.Parse(http.TimeFormat, w.Header().Get("X-RateLimit-Reset"))
	if err != nil {
		t.Fatalf("invalid X-RateLimit-Reset: %v", err)
	}
	if until := time.Until(reset); until > time.Second {
		t.Errorf("expected reset within 1s, got %s", until)
	}
}

func TestRatelimitIsPerIP(t *testing.T) {
	router := newRatelimitTestRouter(t, 1)

	if w := doRatelimitRequest(router, "192.0.2.1"); w.Code != http.StatusOK {
		t.Fatalf("first IP: expected 200, got %d", w.Code)
	}
	if w := doRatelimitRequest(router, "192.0.2.1"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("first IP: expected 429, got %d", w.Code)
	}
	if w := doRatelimitRequest(router, "192.0.2.2"); w.Code != http.StatusOK {
		t.Fatalf("second IP should have its own bucket, got %d", w.Code)
	}
}

func TestRatelimitRateConversion(t *testing.T) {
	tests := []struct {
		rate      float64
		wantLimit string
	}{
		{rate: 0, wantLimit: "10"}, // falls back to default
		{rate: -5, wantLimit: "10"},
		{rate: 0.5, wantLimit: "1"},
		{rate: 2.4, wantLimit: "2"},
		{rate: 1000, wantLimit: "1000"},
	}

	for _, tt := range tests {
		t.Run(strconv.FormatFloat(tt.rate, 'f', -1, 64), func(t *testing.T) {
			w := doRatelimitRequest(newRatelimitTestRouter(t, tt.rate), "192.0.2.1")
			if got := w.Header().Get("X-RateLimit-Limit"); got != tt.wantLimit {
				t.Errorf("expected limit %s, got %s", tt.wantLimit, got)
			}
		})
	}
}
