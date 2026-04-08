package middleware

import (
	"fmt"
	"net/http"

	"saiij.distributed.rate.limiter/internal/ratelimiter"
)

func RateLimiterMiddleware(rateLimiter ratelimiter.RateLimiter, keyExtractor func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyExtractor(r)
			result, err := rateLimiter.Allow(r.Context(), key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.RequestRemain))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%.0f", result.RetryIn.Seconds()))

			if !result.CanAccess {
				w.Header().Set("Retry-After", result.ResetRequestAt.UTC().Format(http.TimeFormat))
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
