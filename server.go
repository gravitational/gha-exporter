package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RunServer(cfg *CLI) error {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	s := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	return s.ListenAndServe()
}
