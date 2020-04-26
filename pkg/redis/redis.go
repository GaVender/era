package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/opentracing/opentracing-go"

	"github.com/GaVender/era/pkg/log"
)

type (
	Config struct {
		Addr         string
		Password     string
		DB           int
		MaxRetries   int
		DialTimeout  int
		ReadTimeout  int
		WriteTimeout int
		PoolSize     int
		MinIdleConns int
		IdleTimeout  int
	}

	Redis struct {
		*redis.Client
		logger          log.Logger
		tracer          opentracing.Tracer
		ableMonitor     bool
		monitorInterval time.Duration
		close           chan bool
	}

	hook struct {
		tracer      opentracing.Tracer
		logger      log.Logger
		beginTime   time.Time
		ableMonitor bool
	}

	Option func(*Redis)
)

const (
	operationProc     = "redis: "
	operationProcPipe = "redis: pipeline: "
)

func NewClient(cfg Config, opts ...Option) (Redis, func()) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  time.Millisecond * time.Duration(cfg.DialTimeout),
		ReadTimeout:  time.Millisecond * time.Duration(cfg.ReadTimeout),
		WriteTimeout: time.Millisecond * time.Duration(cfg.WriteTimeout),
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		IdleTimeout:  time.Millisecond * time.Duration(cfg.IdleTimeout),
	})
	if err := client.Ping().Err(); err != nil {
		panic("redis init: " + err.Error())
	}

	r := Redis{
		Client: client,
		close:  make(chan bool),
	}
	for _, opt := range opts {
		opt(&r)
	}

	if r.logger == nil {
		r.logger = log.NullLogger{}
	}

	r.Client.AddHook(&hook{
		tracer:      r.tracer,
		logger:      r.logger,
		ableMonitor: r.ableMonitor,
	})

	return r, func() {
		if err := client.Close(); err != nil {
			close(r.close)
			r.logger.Errorf("redis close: %s", err.Error())
		}
	}
}

func WithLogger(logger log.Logger) Option {
	return func(r *Redis) {
		r.logger = logger
	}
}

func WithTracer(tracer opentracing.Tracer) Option {
	return func(r *Redis) {
		r.tracer = tracer
	}
}

func WithMonitor(able bool, interval time.Duration) Option {
	return func(r *Redis) {
		r.ableMonitor = able
		r.monitorInterval = interval
	}
}

func (r Redis) PerformanceStats() {
	if !r.ableMonitor {
		return
	}

	r.logger.Infof("redis performance statistics begin...\n")

	go func() {
		defer func() {
			if err := recover(); err != nil {
				r.logger.Errorf("redis performance stats: %v", err)
				r.PerformanceStats()
			}
		}()

		ticker := time.NewTicker(r.monitorInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := r.Client.PoolStats()

				metricsRedisStatsGauge.WithLabelValues("total conns").Set(float64(stats.TotalConns))
				metricsRedisStatsGauge.WithLabelValues("idle conns").Set(float64(stats.IdleConns))
				metricsRedisStatsGauge.WithLabelValues("stale conns").Set(float64(stats.StaleConns))
				metricsRedisStatsGauge.WithLabelValues("hits").Set(float64(stats.Hits))
				metricsRedisStatsGauge.WithLabelValues("misses").Set(float64(stats.Misses))
				metricsRedisStatsGauge.WithLabelValues("timeout").Set(float64(stats.Timeouts))
			case <-r.close:
				r.logger.Infof("redis stats stop...")
				return
			}
		}
	}()
}

func (h *hook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	h.beginTime = time.Now()
	return ctx, nil
}

func (h *hook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	duration := time.Now().Sub(h.beginTime).Milliseconds()
	operationInfo := operationProc + " " + cmd.String()

	if h.tracer != nil {
		rsp := h.tracer.StartSpan(
			operationInfo,
			opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
			opentracing.StartTime(h.beginTime),
		).SetTag("error", cmd.Err())
		rsp.Finish()
	}

	if h.ableMonitor {
		metricsRedisCmdCounter.WithLabelValues(cmd.String()).Inc()
		metricsRedisDurationHistogram.WithLabelValues(cmd.String()).Observe(float64(duration))
	}

	h.logger.Infof(fmt.Sprint(operationInfo, " , duration: ", duration))
	return nil
}

func (h *hook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	h.beginTime = time.Now()
	return ctx, nil
}

func (h *hook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	for _, cmd := range cmds {
		duration := time.Now().Sub(h.beginTime).Milliseconds()
		operationInfo := operationProcPipe + " " + cmd.String()

		if h.tracer != nil {
			rsp := h.tracer.StartSpan(
				operationInfo,
				opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
				opentracing.StartTime(h.beginTime),
			).SetTag("error", cmd.Err())
			rsp.Finish()
		}

		if h.ableMonitor {
			metricsRedisCmdCounter.WithLabelValues(cmd.String()).Inc()
			metricsRedisDurationHistogram.WithLabelValues(cmd.String()).Observe(float64(duration))
		}

		h.logger.Infof(fmt.Sprint(operationInfo, " , duration: ", duration))
	}

	return nil
}
