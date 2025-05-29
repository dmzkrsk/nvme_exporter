package main

import "github.com/prometheus/client_golang/prometheus"

var (
	loopRuns = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "loop_runs_total",
			Help: "Total number of main loop runs",
		},
	)
)

func init() {
	prometheus.MustRegister(
		loopRuns,
	)
}
