package httptransport

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimitBucket struct {
	windowStart time.Time
	count       int
}

func NewRateLimitMiddleware(limit int, window time.Duration, next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	if limit <= 0 {
		return next
	}
	if window <= 0 {
		window = time.Minute
	}
	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)
	now := time.Now
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		key := clientIP(request)
		current := now()
		mu.Lock()
		bucket := buckets[key]
		if bucket.windowStart.IsZero() || current.Sub(bucket.windowStart) >= window {
			bucket = rateLimitBucket{windowStart: current}
		}
		bucket.count++
		buckets[key] = bucket
		limited := bucket.count > limit
		mu.Unlock()
		if limited {
			response.Header().Set("Retry-After", "60")
			http.Error(response, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func clientIP(request *http.Request) string {
	forwarded := strings.TrimSpace(request.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		if first, _, ok := strings.Cut(forwarded, ","); ok {
			return strings.TrimSpace(first)
		}
		return forwarded
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(request.RemoteAddr)
}
