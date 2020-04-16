package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/opentracing/opentracing-go"

	"github.com/GaVender/era/pkg/log"
)

type (
	DB struct {
		*gorm.DB
	}

	Config struct {
		Conn        string
		MaxLifeTime int
		MaxIdleConn int
		MaxOpenConn int
	}
)

const (
	keyBegin = "begin"
	keyCtx   = "ctx"

	traceOperationQuery    = "sql: query : "
	traceOperationRowQuery = "sql: row query : "
	traceOperationCreate   = "sql: create : "
	traceOperationUpdate   = "sql: update : "
	traceOperationDelete   = "sql: delete : "
)

func NewConnection(cfg Config, tracer opentracing.Tracer, log log.Logger) (DB, func()) {
	db, err := gorm.Open("mysql", cfg.Conn)
	if err != nil {
		panic("mysql init: " + err.Error())
	}

	db.SetLogger(log)
	db.SingularTable(true)
	db.BlockGlobalUpdate(true)

	db.DB().SetConnMaxLifetime(time.Millisecond * time.Duration(cfg.MaxLifeTime))
	db.DB().SetMaxIdleConns(cfg.MaxIdleConn)
	db.DB().SetMaxOpenConns(cfg.MaxOpenConn)

	scopeBegin := func(scope *gorm.Scope) {
		scope.Set(keyBegin, time.Now())
	}
	scopeTrace := func(scope *gorm.Scope, t opentracing.Tracer, operation string) {
		beginTime, ok := scope.Get(keyBegin)
		if !ok {
			beginTime = time.Now()
		}

		var duration int64
		if bt, ok := beginTime.(time.Time); ok {
			duration = time.Now().Sub(bt).Milliseconds()
		}

		operationInfo := fmt.Sprint(operation, scope.SQL, " , args: ", scope.SQLVars, " , rowsAffected: ", scope.DB().RowsAffected,
			" , error: ", scope.DB().Error)
		defer log.Infof(fmt.Sprint(operationInfo, " , duration: ", duration))

		if oldCtx, ok := scope.Get(keyCtx); ok {
			if ctx, ok := oldCtx.(context.Context); ok {
				sqlSp := opentracing.StartSpan(
					operationInfo,
					opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
					opentracing.StartTime(beginTime.(time.Time)),
				)
				sqlSp.Finish()
			} else {
				log.Errorf("mysql transform trace ctx fail: %v", oldCtx)
			}
		} else {
			log.Error("mysql get trace ctx fail")
		}
	}

	db.Callback().Query().Before("gorm:query").Register("query-before-1", func(scope *gorm.Scope) {
		scopeBegin(scope)
	})
	db.Callback().Query().After("gorm:query").Register("query-after-1", func(scope *gorm.Scope) {
		scopeTrace(scope, tracer, traceOperationQuery)
	})

	db.Callback().RowQuery().Before("gorm:row_query").Register("row-query-before-1", func(scope *gorm.Scope) {
		scopeBegin(scope)
	})
	db.Callback().RowQuery().After("gorm:row_query").Register("row-query-after-1", func(scope *gorm.Scope) {
		scopeTrace(scope, tracer, traceOperationRowQuery)
	})

	db.Callback().Create().Before("gorm:create").Register("create-before-1", func(scope *gorm.Scope) {
		scopeBegin(scope)
	})
	db.Callback().Create().After("gorm:create").Register("create-after-1", func(scope *gorm.Scope) {
		scopeTrace(scope, tracer, traceOperationCreate)
	})

	db.Callback().Update().Before("gorm:update").Register("update-before-1", func(scope *gorm.Scope) {
		scopeBegin(scope)
	})
	db.Callback().Update().After("gorm:update").Register("update-after-1", func(scope *gorm.Scope) {
		scopeTrace(scope, tracer, traceOperationUpdate)
	})

	db.Callback().Delete().Before("gorm:delete").Register("delete-before-1", func(scope *gorm.Scope) {
		scopeBegin(scope)
	})
	db.Callback().Delete().After("gorm:delete").Register("delete-after-1", func(scope *gorm.Scope) {
		scopeTrace(scope, tracer, traceOperationDelete)
	})

	return DB{
			db,
		}, func() {
			if err = db.Close(); err != nil {
				log.Errorf("mysql close: %s", err.Error())
			}
		}
}

func (db DB) WithContext(ctx context.Context) DB {
	return DB{
		db.Set(keyCtx, ctx),
	}
}
