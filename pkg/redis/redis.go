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
	Redis struct {
		*redis.Client
	}

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

	hook struct {
		tracer    opentracing.Tracer
		log       log.Logger
		beginTime time.Time
	}
)

const (
	operationProc     = "redis: "
	operationProcPipe = "redis: pipeline: "
)

func NewConnection(cfg Config, tracer opentracing.Tracer, log log.Logger) (Redis, func()) {
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

	client.AddHook(&hook{
		tracer: tracer,
		log:    log,
	})

	return Redis{
			client,
		}, func() {
			if err := client.Close(); err != nil {
				log.Errorf("redis close: %s", err.Error())
			}
		}
}

func (h *hook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	h.beginTime = time.Now()
	return ctx, nil
}

func (h *hook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	duration := time.Now().Sub(h.beginTime).Milliseconds()
	operationInfo := operationProc + " " + cmd.String()

	rsp := opentracing.StartSpan(
		operationInfo,
		opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
		opentracing.StartTime(h.beginTime),
	).
		SetTag("error", cmd.Err())
	rsp.Finish()

	h.log.Infof(fmt.Sprint(operationInfo, " , duration: ", duration))
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

		rsp := opentracing.StartSpan(
			operationInfo,
			opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
			opentracing.StartTime(h.beginTime),
		).
			SetTag("error", cmd.Err())
		rsp.Finish()

		h.log.Infof(fmt.Sprint(operationInfo, " , duration: ", duration))
	}

	return nil
}
