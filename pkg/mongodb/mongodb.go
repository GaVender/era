package mongodb

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/GaVender/era/pkg/log"
)

type (
	Mongo struct {
		*mongo.Client
	}

	Config struct {
		App             string
		URI             string
		MaxConnIdleTime int
		SocketTimeout   int
		MaxPoolSize     uint64
		MinPoolSize     uint64
	}
)

func NewConnection(cfg Config, tracer opentracing.Tracer, log log.Logger) (Mongo, func()) {
	idleTime := time.Duration(cfg.MaxConnIdleTime)
	socketTimeout := time.Duration(cfg.SocketTimeout)

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.URI), &options.ClientOptions{
		AppName:         &cfg.App,
		MaxConnIdleTime: &idleTime,
		MaxPoolSize:     &cfg.MaxPoolSize,
		MinPoolSize:     &cfg.MinPoolSize,
		SocketTimeout:   &socketTimeout,
	})
	if err != nil {
		panic("mongodb init: " + err.Error())
	}

	ctx := context.Background()
	/*if err = client.Ping(ctx, readpref.Primary()); err != nil {
		panic("mongodb ping: " + err.Error())
	}*/

	return Mongo{
			client,
		}, func() {
			if err = client.Disconnect(ctx); err != nil {
				log.Errorf("mongodb close: %s", err.Error())
			}
		}
}
