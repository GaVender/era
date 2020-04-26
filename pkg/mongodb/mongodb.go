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
	Config struct {
		App             string
		Hosts           []string
		MaxConnIdleTime int
		MaxPoolSize     uint64
		MinPoolSize     uint64
	}

	Mongo struct {
		*mongo.Client
		logger          log.Logger
		tracer          opentracing.Tracer
		ableMonitor     bool
		monitorInterval time.Duration
		close           chan bool
	}

	hook struct {
		tracer      opentracing.Tracer
		logger      log.Logger
		ableMonitor bool
	}

	Option func(*Mongo)
)

const (
	operation = "mongo: "
)

var (
	startTime  sync.Map
	startEvent sync.Map
)

func NewConnection(cfg Config, opts ...Option) (Mongo, func()) {
	m := Mongo{
		close: make(chan bool),
	}

	for _, opt := range opts {
		opt(&m)
	}

	idleTime := time.Duration(cfg.MaxConnIdleTime)
	h := hook{
		tracer:      m.tracer,
		logger:      m.logger,
		ableMonitor: m.ableMonitor,
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
		options.Client().SetPoolMonitor(&event.PoolMonitor{
			Event: h.poolMonitor(),
		}),
	)
	if err != nil {
		panic("mongodb init: " + err.Error())
	}

	ctx := context.Background()
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		panic("mongodb ping: " + err.Error())
	}

	m.Client = client
	return m, func() {
		if err = client.Disconnect(ctx); err != nil {
			close(m.close)
			m.logger.Errorf("mongodb close: %s", err.Error())
		}
	}
}

func WithLogger(logger log.Logger) Option {
	return func(m *Mongo) {
		m.logger = logger
	}
}

func WithTracer(tracer opentracing.Tracer) Option {
	return func(m *Mongo) {
		m.tracer = tracer
	}
}

func WithMonitor(able bool, interval time.Duration) Option {
	return func(m *Mongo) {
		m.ableMonitor = able
		m.monitorInterval = interval
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
		unknownEvent, ok1 := startEvent.Load(succeededEvent.RequestID)
		startedEvent, ok2 := unknownEvent.(*event.CommandStartedEvent)
		if !ok1 || !ok2 {
			startedEvent = &event.CommandStartedEvent{}
		}

		operationInfo := operation + succeededEvent.CommandName

		if h.tracer != nil {
			msp := h.tracer.StartSpan(
				operationInfo,
				opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
				opentracing.StartTime(startedTime.(time.Time)),
			).
				SetTag("mongodb request id", succeededEvent.RequestID).
				SetTag("command", startedEvent.Command).
				SetTag("result", succeededEvent.Reply.String())
			msp.Finish()
		}

		if h.ableMonitor {
			metricsMongodbQueryCounter.WithLabelValues(startedEvent.DatabaseName, succeededEvent.CommandName, "success").Inc()
			metricsMongodbDurationHistogram.WithLabelValues(startedEvent.DatabaseName, startedEvent.Command.String()).
				Observe(float64(succeededEvent.DurationNanos / 1e6))
		}

		h.logger.Infof(fmt.Sprint(operationInfo, " , duration: ", succeededEvent.DurationNanos/1e6))
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
		unknownEvent, ok1 := startEvent.Load(failedEvent.RequestID)
		startedEvent, ok2 := unknownEvent.(*event.CommandStartedEvent)
		if !ok1 || !ok2 {
			startedEvent = &event.CommandStartedEvent{}
		}

		operationInfo := operation + failedEvent.CommandName

		if h.tracer != nil {
			msp := h.tracer.StartSpan(
				operationInfo,
				opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
				opentracing.StartTime(startedTime.(time.Time)),
			).
				SetTag("mongodb request id", failedEvent.RequestID).
				SetTag("command", startedEvent.Command).
				SetTag("error", failedEvent.Failure)
			msp.Finish()
		}

		if h.ableMonitor {
			metricsMongodbQueryCounter.WithLabelValues(startedEvent.DatabaseName, failedEvent.CommandName, "fail").Inc()
			metricsMongodbDurationHistogram.WithLabelValues(startedEvent.DatabaseName, startedEvent.Command.String()).
				Observe(float64(failedEvent.DurationNanos / 1e6))
		}

		h.logger.Errorf(fmt.Sprint(operationInfo, " , duration: ", failedEvent.DurationNanos/1e6))
		startTime.Delete(failedEvent.RequestID)
		startEvent.Delete(failedEvent.RequestID)
	}
}

func (h hook) poolMonitor() func(poolEvent *event.PoolEvent) {
	return func(poolEvent *event.PoolEvent) {
		if h.ableMonitor {
			metricsMongodbStatsGauge.WithLabelValues(poolEvent.Address, poolEvent.Type).Inc()
		}
	}
}
