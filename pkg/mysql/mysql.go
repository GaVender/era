package mysql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/opentracing/opentracing-go"

	"github.com/GaVender/era/pkg/log"
)

type (
	Config struct {
		Conn        string
		DBName      string
		MaxLifeTime int
		MaxIdleConn int
		MaxOpenConn int
	}

	Client struct {
		*gorm.DB
		logger          log.Logger
		tracer          opentracing.Tracer
		db              string
		ableMonitor     bool
		monitorInterval time.Duration
		close           chan bool
	}

	Option func(db *Client)
)

const (
	keyBegin = "begin"
	keyCtx   = "ctx"

	operation = "mysql: "
)

func NewClient(cfg Config, opts ...Option) (Client, func()) {
	if len(cfg.DBName) == 0 {
		panic("mysql config without db name")
	}

	db, err := gorm.Open("mysql", cfg.Conn)
	if err != nil {
		panic("mysql init: " + err.Error())
	}

	db.SingularTable(true)
	db.BlockGlobalUpdate(true)
	db.DB().SetConnMaxLifetime(time.Millisecond * time.Duration(cfg.MaxLifeTime))
	db.DB().SetMaxIdleConns(cfg.MaxIdleConn)
	db.DB().SetMaxOpenConns(cfg.MaxOpenConn)

	client := Client{
		db:    cfg.DBName,
		close: make(chan bool),
	}

	for _, opt := range opts {
		opt(&client)
	}

	if client.logger == nil {
		client.logger = log.NullLogger{}
	}
	db.SetLogger(client.logger)
	client.DB = db

	scopeBegin := func(scope *gorm.Scope) {
		scope.Set(keyBegin, time.Now())
	}
	scopeTrace := func(scope *gorm.Scope) {
		beginTime, ok := scope.Get(keyBegin)
		if !ok {
			beginTime = time.Now()
		}

		var duration time.Duration
		if bt, ok := beginTime.(time.Time); ok {
			duration = time.Now().Sub(bt)
		}

		ctx := context.Background()
		operationInfo := fmt.Sprint(operation, strings.ToLower(scope.SQL[0:strings.Index(scope.SQL, " ")]))

		if client.tracer != nil {
			if oldCtx, ok := scope.Get(keyCtx); ok {
				ctx, ok = oldCtx.(context.Context)
				if ok {
					var childSp opentracing.Span
					sp := opentracing.SpanFromContext(ctx)
					if sp == nil {
						childSp, ctx = opentracing.StartSpanFromContextWithTracer(
							ctx,
							client.tracer,
							operationInfo,
							opentracing.StartTime(beginTime.(time.Time)),
						)
					} else {
						childSp = client.tracer.StartSpan(
							operationInfo,
							opentracing.ChildOf(sp.Context()),
							opentracing.StartTime(beginTime.(time.Time)),
						)
					}

					childSp.SetTag("sql", scope.SQL).
						SetTag("args", scope.SQLVars).
						SetTag("rowsAffected", scope.DB().RowsAffected).
						SetTag("error", scope.DB().Error)
					childSp.Finish()
				} else {
					client.logger.ContextErrorf(ctx, "mysql transform trace ctx fail: %v", oldCtx)
				}
			} else {
				client.logger.ContextErrorf(ctx, "mysql get trace ctx fail")
			}
		}

		if client.ableMonitor {
			metricsMysqlQueryCounter.WithLabelValues(client.db, scope.SQL).Inc()
			metricsMysqlDurationHistogram.WithLabelValues(client.db, fmt.Sprint(scope.SQL, scope.SQLVars)).
				Observe(float64(duration.Milliseconds()))
		}

		client.logger.ContextInfof(ctx, fmt.Sprint(operation, "sql: ", scope.SQL, " , args: ", scope.SQLVars,
			" , duration: ", duration.Milliseconds()))
	}

	db.Callback().Query().Before("gorm:query").Register("query-before-1", func(scope *gorm.Scope) {
		scopeBegin(scope)
	})
	db.Callback().Query().After("gorm:query").Register("query-after-1", func(scope *gorm.Scope) {
		scopeTrace(scope)
	})

	db.Callback().RowQuery().Before("gorm:row_query").Register("row-query-before-1", func(scope *gorm.Scope) {
		scopeBegin(scope)
	})
	db.Callback().RowQuery().After("gorm:row_query").Register("row-query-after-1", func(scope *gorm.Scope) {
		scopeTrace(scope)
	})

	db.Callback().Create().Before("gorm:create").Register("create-before-1", func(scope *gorm.Scope) {
		scopeBegin(scope)
	})
	db.Callback().Create().After("gorm:create").Register("create-after-1", func(scope *gorm.Scope) {
		scopeTrace(scope)
	})

	db.Callback().Update().Before("gorm:update").Register("update-before-1", func(scope *gorm.Scope) {
		scopeBegin(scope)
	})
	db.Callback().Update().After("gorm:update").Register("update-after-1", func(scope *gorm.Scope) {
		scopeTrace(scope)
	})

	db.Callback().Delete().Before("gorm:delete").Register("delete-before-1", func(scope *gorm.Scope) {
		scopeBegin(scope)
	})
	db.Callback().Delete().After("gorm:delete").Register("delete-after-1", func(scope *gorm.Scope) {
		scopeTrace(scope)
	})

	return client, func() {
		if err = db.Close(); err != nil {
			close(client.close)
			client.logger.Errorf("mysql close: %s", err.Error())
		}
	}
}

func WithLogger(logger log.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

func WithTracer(tracer opentracing.Tracer) Option {
	return func(c *Client) {
		c.tracer = tracer
	}
}

func WithMonitor(able bool, interval time.Duration) Option {
	return func(c *Client) {
		c.ableMonitor = able
		c.monitorInterval = interval
	}
}

func (c Client) WithContext(ctx context.Context) Client {
	c.DB = c.DB.Set(keyCtx, ctx)
	return c
}

func (c Client) PerformanceStats() {
	if !c.ableMonitor {
		return
	}

	c.logger.Infof("mysql performance statistics begin...")

	go func() {
		defer func() {
			if err := recover(); err != nil {
				c.logger.Errorf("mysql performance stats: %v", err)
				c.PerformanceStats()
			}
		}()

		ticker := time.NewTicker(c.monitorInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := c.DB.DB().Stats()

				metricsMysqlStatsGauge.WithLabelValues(c.db, "max open conn").Set(float64(stats.MaxOpenConnections))
				metricsMysqlStatsGauge.WithLabelValues(c.db, "open conn").Set(float64(stats.OpenConnections))
				metricsMysqlStatsGauge.WithLabelValues(c.db, "in use conn").Set(float64(stats.InUse))
				metricsMysqlStatsGauge.WithLabelValues(c.db, "idle conn").Set(float64(stats.Idle))
				metricsMysqlStatsGauge.WithLabelValues(c.db, "wait count").Set(float64(stats.WaitCount))
				metricsMysqlStatsGauge.WithLabelValues(c.db, "wait duration").Set(float64(stats.WaitDuration))
				metricsMysqlStatsGauge.WithLabelValues(c.db, "max lifetime closed").Set(float64(stats.MaxLifetimeClosed))
			case <-c.close:
				c.logger.Infof("mysql stats stop...")
				return
			}
		}
	}()
}
