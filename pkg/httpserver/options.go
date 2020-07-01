package httpserver

import (
	"context"
	"hash/maphash"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v4/middleware"
	"github.com/go-chi/httplog"
	"github.com/segmentio/ksuid"
)

type ContextValue string

const (
	RequestKSUID ContextValue = "RequestKSUID"
)

var (
	DefaultOptions = []Option{
		WithRecover(),
		WithCompression(),
		WithRealIP(),
		WithRequestKSUID(),
		WithReqLogger(),
	}

	DebugOptions = []Option{
		WithRecover(),
		WithCompression(),
		WithRealIP(),
		WithRequestKSUID(),
		WithProfiler(),
		WithReqLogger(),
		WithIPRatelimit(1, 10, time.Second),
	}
)

type Option func(*HTTPServer)

func WithMiddleware(f func(http.Handler) http.Handler) func(*HTTPServer) {
	return func(w *HTTPServer) {
		w.router = f(w.router)
		w.server.Handler = w.router
	}
}

func WithListenAdr(listenAdr string) func(*HTTPServer) {
	return func(w *HTTPServer) {
		w.listen = listenAdr
		w.server.Addr = w.listen
	}
}

func WithRealIP() func(*HTTPServer) {
	return WithMiddleware(middleware.RealIP)
}

func WithReqLogger() func(*HTTPServer) {
	return func(w *HTTPServer) {
		// TODO: Re-roll with zap instead of zerolog
		logger := httplog.NewLogger("http(s)-request", httplog.Options{
			JSON: true,
		})
		w.router = httplog.RequestLogger(logger)(w.router)
		w.server.Handler = w.router
	}
}

func WithRecover() func(*HTTPServer) {
	return WithMiddleware(middleware.Recoverer)
}

func WithCompression() func(*HTTPServer) {
	return WithMiddleware(middleware.DefaultCompress)
}

func WithRequestKSUID() func(*HTTPServer) {
	return WithMiddleware(requestKSUID)
}

func WithPing() func(*HTTPServer) {
	return WithMiddleware(middleware.Heartbeat("ping"))
}

func WithProfiler() func(*HTTPServer) {
	profiler := middleware.Profiler()
	return func(w *HTTPServer) {
		oldHandler := w.router
		w.router = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/debug") {
				profiler.ServeHTTP(w, r)
			} else {
				oldHandler.ServeHTTP(w, r)
			}
		})
		w.server.Handler = w.router
	}
}

func requestKSUID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = ksuid.New().String()
		}
		ctx = context.WithValue(ctx, RequestKSUID, requestID)
		w.Header().Add("X-Request-Id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// WithIpThrottle will hash each requests RemoteAdr to a bucket which is refilled with tokens
// every $rate duration up till burstLimit. On each request, a token is taken, if not tokens are
// available the request will fail with http.StatusTooManyRequests
// The middleware will NOT cleanup channels and tickers since this is expected to survive the
// full lifecycle of the service
func WithIPRatelimit(buckets, burstLimit uint64, rate time.Duration) func(*HTTPServer) {
	ticker := time.NewTicker(rate)
	throttles := make([]chan bool, buckets)
	for i := range throttles {
		throttles[i] = make(chan bool, burstLimit)
		// Each restart have a fresh burst cache
		for j := uint64(0); j < burstLimit; j++ {
			select {
			case throttles[i] <- true:
			default:
			}
		}
	}

	go func() {
		for range ticker.C {
			for j := uint64(0); j < buckets; j++ {
				select {
				case throttles[j] <- true:
				default:
				}
			}
		}

	}()

	f := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := maphash.Hash{}
			_, err := h.WriteString(r.RemoteAddr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
			}
			bucket := h.Sum64() % buckets
			select {
			case <-throttles[bucket]:
				next.ServeHTTP(w, r)
			default:
				w.WriteHeader(http.StatusTooManyRequests)
			}
		})
	}
	return WithMiddleware(f)
}
