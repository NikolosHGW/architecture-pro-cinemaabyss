package main

import (
	"errors"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func newHandler(cfg config) http.Handler {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 5 * time.Second

	monolith := reverseProxy(cfg.monolithURL, transport)
	movies := reverseProxy(cfg.moviesServiceURL, transport)
	events := reverseProxy(cfg.eventsServiceURL, transport)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, "Proxy is healthy")
	})
	mux.Handle("GET /api/movies/health", movies)
	mux.Handle("GET /api/events/health", events)

	for _, path := range []string{"/api/users", "/api/payments", "/api/subscriptions"} {
		mux.Handle("GET "+path, monolith)
		mux.Handle("POST "+path, monolith)
	}
	for _, path := range []string{"/api/events/movie", "/api/events/user", "/api/events/payment"} {
		mux.Handle("POST "+path, events)
	}

	movieHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cfg.gradualMigration || rand.IntN(100) < cfg.moviesMigrationPercent {
			movies.ServeHTTP(w, r)
			return
		}
		monolith.ServeHTTP(w, r)
	})
	mux.Handle("GET /api/movies", movieHandler)
	mux.Handle("POST /api/movies", movieHandler)
	return mux
}

func reverseProxy(target *url.URL, transport http.RoundTripper) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.SetXForwarded()
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy %s %s: %v", r.Method, r.URL.Path, err)
			status := http.StatusBadGateway
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				status = http.StatusGatewayTimeout
			}
			http.Error(w, http.StatusText(status), status)
		},
	}
}
