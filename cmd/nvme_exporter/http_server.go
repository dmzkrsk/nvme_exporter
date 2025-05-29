package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metricsHTTPServer struct {
	listen string
	srv    *http.Server
}

func newMetricsHTTPServer(listen string) *metricsHTTPServer {
	return &metricsHTTPServer{
		listen: listen,
	}
}

func (s metricsHTTPServer) GetAddr() string {
	return s.listen
}

func (s *metricsHTTPServer) Shutdown() error {
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := s.srv.Shutdown(stopCtx)
	cancel()
	return err
}

func (s *metricsHTTPServer) Run() error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Handle("/metrics", promhttp.Handler())

	r.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<html>
			<head><title>%[1]s</title></head>
			<body>
			<h1>%[1]s</h1>
			<p>Visit <a href="/metrics">/metrics</a> to see metrics about the exporter.</p>
			</body>
			</html>`, appName)
	})

	s.srv = &http.Server{
		Addr:              s.listen,
		Handler:           r,
		ReadHeaderTimeout: time.Second,
	}

	err := s.srv.ListenAndServe()
	if err == nil || err == http.ErrServerClosed {
		return nil
	}

	return err
}
