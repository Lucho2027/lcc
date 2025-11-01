package serverkit

import (
	"expvar"
	"net/http"
	"time"
)

var (
	expTotal    = expvar.NewMap("http_requests_total")
	expInFlight = expvar.NewInt("http_requests_in_flight")
	expLatency  = expvar.NewMap("http_request_duration_ms")
)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		expInFlight.Add(1)
		defer expInFlight.Add(-1)
		method := r.Method
		next.ServeHTTP(w, r)

		duration := time.Since(start).Milliseconds()
		incrMap(expTotal, method, 1)
		incrMap(expLatency, method, duration)
	})

}

func incrMap(m *expvar.Map, key string, delta int64) {
	v := m.Get(key)
	if v == nil {
		v = new(expvar.Int)
		m.Set(key, v)
	}
	v.(*expvar.Int).Add(delta)
}
