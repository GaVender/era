package prometheus

import (
	"errors"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/GaVender/era/pkg/log"
)

type (
	Config struct {
		Disabled bool
		Host     string
	}

	Server struct {
		logger log.Logger
	}

	Option func(*Server)
)

const (
	defaultHost = ":9191"
)

var (
	ErrInvalidHost = errors.New("invalid host")
)

func NewService(cfg Config, opts ...Option) (*Server, error) {
	s := &Server{}

	if !cfg.Disabled {
		if len(cfg.Host) == 0 {
			cfg.Host = defaultHost
		}

		for _, opt := range opts {
			opt(s)
		}

		if s.logger == nil {
			s.logger = log.NullLogger{}
		}

		if strings.Index(cfg.Host, ":") < 0 {
			err := ErrInvalidHost
			return nil, err
		}

		go func() {
			http.Handle("/metrics", promhttp.Handler())
			s.logger.Errorf("prometheus init: %s", http.ListenAndServe(cfg.Host, nil).Error())
		}()
	}

	return s, nil
}

func WithLogger(logger log.Logger) Option {
	return func(server *Server) {
		server.logger = logger
	}
}
