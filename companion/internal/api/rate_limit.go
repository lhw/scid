package api

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type rateBucket struct {
	count   int
	resetAt time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]rateBucket
	ops     uint64
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{buckets: make(map[string]rateBucket)}
}

func (l *rateLimiter) allow(key string, limit int, window time.Duration, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.ops++
	if l.ops%256 == 0 {
		for bucketKey, bucket := range l.buckets {
			if !bucket.resetAt.After(now) {
				delete(l.buckets, bucketKey)
			}
		}
	}

	bucket, ok := l.buckets[key]
	if !ok || !bucket.resetAt.After(now) {
		l.buckets[key] = rateBucket{count: 1, resetAt: now.Add(window)}
		return true, 0
	}

	if bucket.count >= limit {
		return false, bucket.resetAt.Sub(now)
	}

	bucket.count++
	l.buckets[key] = bucket
	return true, 0
}

func (s *Server) publicRateLimitMiddleware(name string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return s.rateLimitMiddleware(name, limit, window, func(r *http.Request) string {
		return clientAddr(r)
	})
}

func (s *Server) authenticatedRateLimitMiddleware(name string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return s.rateLimitMiddleware(name, limit, window, func(r *http.Request) string {
		if user := userFromContext(r.Context()); user != nil && user.ID != "" {
			return user.ID
		}
		return clientAddr(r)
	})
}

func (s *Server) rateLimitMiddleware(name string, limit int, window time.Duration, keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := name + ":" + keyFunc(r)
			allowed, retryAfter := s.limiter.allow(key, limit, window, time.Now())
			if !allowed {
				seconds := int(math.Ceil(retryAfter.Seconds()))
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientAddr(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
