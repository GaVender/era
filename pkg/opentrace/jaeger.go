package opentrace

import (
	"github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"

	"github.com/GaVender/era/pkg/log"
)

func NewTracer(project string, logger log.Logger, cfg jaegercfg.Configuration) (opentracing.Tracer, func()) {
	cfg.ServiceName = project
	tracer, closer, err := cfg.NewTracer(
		jaegercfg.Logger(logger),
	)
	if err != nil {
		panic("tracer init: " + err.Error())
	}

	opentracing.SetGlobalTracer(tracer)
	return tracer, func() {
		if err := closer.Close(); err != nil {
			panic("tracer close: " + err.Error())
		}
	}
}
