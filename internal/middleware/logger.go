package middleware

import (
	"log"
	"net/http"
	"time"
)

type responseWrite struct {
	http.ResponseWriter
	code int
}

func (rw *responseWrite) WriteHeader(code int) {
	rw.code = code
	rw.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWrite{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)

		log.Printf("[LOG] %3d | %10v | %-5s | %s\n",
			rw.code,
			duration,
			r.Method,
			r.Pattern,
		)
	})
}
