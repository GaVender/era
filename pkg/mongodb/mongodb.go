package mongodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/GaVender/era/pkg/log"
)

type (
	Mongo struct {
		*mongo.Client
	}

	Config struct {
		App             string
		Hosts           []string
		MaxConnIdleTime int
		MaxPoolSize     uint64
		MinPoolSize     uint64
	}

	hook struct {
		tracer       opentracing.Tracer
		log          log.Logger
		beginTime    map[int64]time.Time
		startedEvent map[int64]*event.CommandStartedEvent
		sync.Mutex
	}
)

const (
	operation = "mongo: "
)

func NewConnection(cfg Config, tracer opentracing.Tracer, log log.Logger) (Mongo, func()) {
	idleTime := time.Duration(cfg.MaxConnIdleTime)
	h := hook{
		tracer:       tracer,
		log:          log,
		beginTime:    make(map[int64]time.Time),
		startedEvent: make(map[int64]*event.CommandStartedEvent),
	}

	client, err := mongo.Connect(
		context.Background(),
		&options.ClientOptions{
			AppName:         &cfg.App,
			Hosts:           cfg.Hosts,
			MaxConnIdleTime: &idleTime,
			MaxPoolSize:     &cfg.MaxPoolSize,
			MinPoolSize:     &cfg.MinPoolSize,
		},
		options.Client().SetMonitor(&event.CommandMonitor{
			Started:   h.start(),
			Succeeded: h.succeed(),
			Failed:    h.fail(),
		}),
	)
	if err != nil {
		panic("mongodb init: " + err.Error())
	}

	ctx := context.Background()
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		panic("mongodb ping: " + err.Error())
	}

	return Mongo{
			client,
		}, func() {
			if err = client.Disconnect(ctx); err != nil {
				log.Errorf("mongodb close: %s", err.Error())
			}
		}
}

func (h hook) start() func(context.Context, *event.CommandStartedEvent) {
	return func(ctx context.Context, startedEvent *event.CommandStartedEvent) {
		h.Lock()
		h.beginTime[startedEvent.RequestID] = time.Now()
		h.startedEvent[startedEvent.RequestID] = startedEvent
		h.Unlock()
	}
}

func (h hook) succeed() func(context.Context, *event.CommandSucceededEvent) {
	return func(ctx context.Context, succeededEvent *event.CommandSucceededEvent) {
		if succeededEvent.CommandName == "ping" || succeededEvent.CommandName == "endSessions" {
			return
		}

		h.Lock()
		operationInfo := operation + succeededEvent.CommandName

		msp := h.tracer.StartSpan(
			operationInfo,
			opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
			opentracing.StartTime(h.beginTime[succeededEvent.RequestID]),
		).
			SetTag("mongodb request id", succeededEvent.RequestID).
			SetTag("command", h.startedEvent[succeededEvent.RequestID].Command).
			SetTag("result", succeededEvent.Reply.String())
		msp.Finish()

		h.log.Infof(fmt.Sprint(operationInfo, " , duration: ", succeededEvent.DurationNanos/1e6))
		delete(h.beginTime, succeededEvent.RequestID)
		delete(h.startedEvent, succeededEvent.RequestID)
		h.Unlock()
	}
}

func (h hook) fail() func(context.Context, *event.CommandFailedEvent) {
	return func(ctx context.Context, failedEvent *event.CommandFailedEvent) {
		if failedEvent.CommandName == "ping" || failedEvent.CommandName == "endSessions" {
			return
		}

		h.Lock()
		operationInfo := operation + failedEvent.CommandName

		msp := h.tracer.StartSpan(
			operationInfo,
			opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
			opentracing.StartTime(h.beginTime[failedEvent.RequestID]),
		).
			SetTag("mongodb request id", failedEvent.RequestID).
			SetTag("command", h.startedEvent[failedEvent.RequestID].Command).
			SetTag("error", failedEvent.Failure)
		msp.Finish()

		h.log.Errorf(fmt.Sprint(operationInfo, " , duration: ", failedEvent.DurationNanos/1e6))
		delete(h.beginTime, failedEvent.RequestID)
		delete(h.startedEvent, failedEvent.RequestID)
		h.Unlock()
	}
}
