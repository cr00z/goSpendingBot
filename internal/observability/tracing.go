package observability

import (
	"github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

func InitTracing(logger *zap.Logger) {
	cfg := config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LocalAgentHostPort: "host.docker.internal:6831",
		},
	}
	_, err := cfg.InitGlobalTracer("gospend-bot")
	if err != nil {
		logger.Fatal("cannot init tracing", zap.Error(err))
	}
}
