package observability

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	RequestsCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ozon",
			Subsystem: "telegram",
			Name:      "requests_total",
		},
		[]string{"command"},
	)
	HistogramCommandTimeVec = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ozon",
			Subsystem: "telegram",
			Name:      "histogram_command_time_vec_seconds",
			Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2},
		},
		[]string{"command"},
	)
)

type MetricsServer struct {
	srv    *http.Server
	logger *zap.Logger
}

func NewMetricsServer(logger *zap.Logger) *MetricsServer {
	return &MetricsServer{
		srv:    &http.Server{Addr: ":8080"},
		logger: logger,
	}
}

func (ms *MetricsServer) Start() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := ms.srv.ListenAndServe()
		if err != http.ErrServerClosed {
			ms.logger.Fatal("error starting metrics server", zap.Error(err))
		}
	}()
}

func (ms *MetricsServer) Stop() {
	if err := ms.srv.Shutdown(context.TODO()); err != nil {
		ms.logger.Fatal(
			"failure/timeout shutting down the server gracefully",
			zap.Error(err),
		)
	}
}
