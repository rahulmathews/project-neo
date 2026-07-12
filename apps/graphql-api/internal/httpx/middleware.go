package httpx

import (
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"project-neo/graphql-api/internal/metrics"

	"github.com/rs/cors"
	"golang.org/x/time/rate"
)

// Recover catches panics in downstream handlers, logs the stack, and returns 500.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rv := recover(); rv != nil {
					logger.Error(
						"panic in handler",
						"path", r.URL.Path,
						"method", r.Method,
						"panic", rv,
						"stack", string(debug.Stack()),
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// RequestLog logs method, path, status, duration for each request.
// Skips routes that match any prefix in skipPrefixes (e.g. health checks).
func RequestLog(logger *slog.Logger, skipPrefixes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range skipPrefixes {
				if strings.HasPrefix(r.URL.Path, p) {
					next.ServeHTTP(w, r)
					return
				}
			}
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			logger.Info(
				"request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote", clientIP(r),
			)
		})
	}
}

// BodyLimit caps request body size. Returns 413 if exceeded.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS configures cross-origin handling. allowedOrigins supports "*" or a list.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions, http.MethodDelete},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           int((12 * time.Hour).Seconds()),
	})
	return c.Handler
}

type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*entry
	rps      rate.Limit
	burst    int
	ttl      time.Duration
}

type entry struct {
	lim  *rate.Limiter
	seen time.Time
}

// RateLimit returns per-IP token-bucket middleware. rps is requests/second steady-state.
// burst is the max tokens accumulated. Idle limiters are evicted after ttl.
func RateLimit(rps, burst int) func(http.Handler) http.Handler {
	il := &ipLimiter{
		limiters: make(map[string]*entry),
		rps:      rate.Limit(rps),
		burst:    burst,
		ttl:      10 * time.Minute,
	}
	go il.gc()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lim := il.get(clientIP(r))
			if !lim.Allow() {
				retryAfter := time.Second
				w.Header().Set("Retry-After", strconv.FormatFloat(retryAfter.Seconds(), 'f', 0, 64))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (il *ipLimiter) get(ip string) *rate.Limiter {
	il.mu.Lock()
	defer il.mu.Unlock()
	e, ok := il.limiters[ip]
	if !ok {
		e = &entry{lim: rate.NewLimiter(il.rps, il.burst)}
		il.limiters[ip] = e
	}
	e.seen = time.Now()
	return e.lim
}

func (il *ipLimiter) gc() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		cutoff := time.Now().Add(-il.ttl)
		il.mu.Lock()
		for ip, e := range il.limiters {
			if e.seen.Before(cutoff) {
				delete(il.limiters, ip)
			}
		}
		il.mu.Unlock()
	}
}

// clientIP extracts the request's IP, honoring X-Forwarded-For if present.
// Trust the first hop only; production must run behind a trusted reverse proxy.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Metrics records HTTP request count, duration, and in-flight gauge for a known route label.
// Pass the route label (e.g. "/query") rather than r.URL.Path to keep cardinality bounded.
func Metrics(m *metrics.HTTP, route string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m.InFlight.Inc()
			defer m.InFlight.Dec()
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			m.RequestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(status)).Inc()
			m.RequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		})
	}
}

// Chain composes middlewares in order: Chain(a, b, c)(h) == a(b(c(h))).
func Chain(handler http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}
