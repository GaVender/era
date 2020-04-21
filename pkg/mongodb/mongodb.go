package mongodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.mongodb.org/mongo-driver/bson"
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
		tracer opentracing.Tracer
		log    log.Logger
	}
)

const (
	operation = "mongo: "
)

var (
	startTime  sync.Map
	startEvent sync.Map
)

func NewConnection(cfg Config, tracer opentracing.Tracer, log log.Logger) (Mongo, func()) {
	idleTime := time.Duration(cfg.MaxConnIdleTime)
	h := hook{
		tracer: tracer,
		log:    log,
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
		startTime.Store(startedEvent.RequestID, time.Now())
		startEvent.Store(startedEvent.RequestID, startedEvent)
	}
}

func (h hook) succeed() func(context.Context, *event.CommandSucceededEvent) {
	return func(ctx context.Context, succeededEvent *event.CommandSucceededEvent) {
		if succeededEvent.CommandName == "ping" || succeededEvent.CommandName == "endSessions" {
			return
		}

		startedTime, ok := startTime.Load(succeededEvent.RequestID)
		if !ok {
			startedTime = time.Now()
		}
		startedEvent, ok := startEvent.Load(succeededEvent.RequestID)
		command := bson.Raw{}
		if ok {
			if _, ok := startedEvent.(*event.CommandStartedEvent); ok {
				command = startedEvent.(*event.CommandStartedEvent).Command
			}
		}

		operationInfo := operation + succeededEvent.CommandName

		msp := h.tracer.StartSpan(
			operationInfo,
			opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
			opentracing.StartTime(startedTime.(time.Time)),
		).
			SetTag("mongodb request id", succeededEvent.RequestID).
			SetTag("command", command).
			SetTag("result", succeededEvent.Reply.String())
		msp.Finish()

		h.log.Infof(fmt.Sprint(operationInfo, " , duration: ", succeededEvent.DurationNanos/1e6))
		startTime.Delete(succeededEvent.RequestID)
		startEvent.Delete(succeededEvent.RequestID)
	}
}

func (h hook) fail() func(context.Context, *event.CommandFailedEvent) {
	return func(ctx context.Context, failedEvent *event.CommandFailedEvent) {
		if failedEvent.CommandName == "ping" || failedEvent.CommandName == "endSessions" {
			return
		}

		startedTime, ok := startTime.Load(failedEvent.RequestID)
		if !ok {
			startedTime = time.Now()
		}
		startedEvent, ok := startEvent.Load(failedEvent.RequestID)
		command := bson.Raw{}
		if ok {
			if _, ok := startedEvent.(*event.CommandStartedEvent); ok {
				command = startedEvent.(*event.CommandStartedEvent).Command
			}
		}

		operationInfo := operation + failedEvent.CommandName

		msp := h.tracer.StartSpan(
			operationInfo,
			opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
			opentracing.StartTime(startedTime.(time.Time)),
		).
			SetTag("mongodb request id", failedEvent.RequestID).
			SetTag("command", command).
			SetTag("error", failedEvent.Failure)
		msp.Finish()

		h.log.Errorf(fmt.Sprint(operationInfo, " , duration: ", failedEvent.DurationNanos/1e6))
		startTime.Delete(failedEvent.RequestID)
		startEvent.Delete(failedEvent.RequestID)
	}
}
