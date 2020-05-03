package httpclient

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/GaVender/cast"
	"github.com/opentracing/opentracing-go"

	"github.com/GaVender/era/pkg/log"
)

type (
	Client struct {
		*cast.Cast
		tracer opentracing.Tracer
		logger log.Logger
	}
)

const (
	operation = "http: "
)

func NewClient(ss ...cast.Setter) (*Client, error) {
	c, err := cast.New(ss...)
	if err != nil {
		return nil, err
	}

	client := Client{
		Cast:   c,
		logger: log.NullLogger{},
	}

	return &client, nil
}

func (c *Client) WithTracer(tracer opentracing.Tracer) *Client {
	c.tracer = tracer
	return c
}

func (c *Client) WithLogger(logger log.Logger) *Client {
	c.logger = logger
	return c
}

func (c *Client) Send(ctx context.Context, request *cast.Request) (resp *cast.Response, err error) {
	beginTime := time.Now()
	urlStr := c.GetBaseURL()
	urlInfo, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	operationInfo := operation + urlInfo.Path

	if c.tracer != nil {
		var childSp opentracing.Span
		sp := opentracing.SpanFromContext(ctx)
		if sp == nil {
			childSp, ctx = opentracing.StartSpanFromContextWithTracer(
				ctx,
				c.tracer,
				operationInfo,
				opentracing.StartTime(beginTime),
			)
		} else {
			childSp = c.tracer.StartSpan(
				operationInfo,
				opentracing.ChildOf(sp.Context()),
				opentracing.StartTime(beginTime),
			)
		}

		carrier := opentracing.HTTPHeadersCarrier(request.GetHeader())
		if err := c.tracer.Inject(childSp.Context(), opentracing.HTTPHeaders, carrier); err != nil {
			c.logger.ContextErrorf(ctx, "http request carrier fail")
		}

		defer func() {
			childSp.SetTag("url", c.GetBaseURL()).
				SetTag("method", request.GetMethod()).
				SetTag("header", request.RawRequest().Header).
				SetTag("query", urlInfo.Query()).
				SetTag("body", string(request.GetBody())).
				SetTag("status code", resp.StatusCode()).
				SetTag("response", string(resp.Body()))
			childSp.Finish()
		}()
	}

	resp, err = c.Do(ctx, request)
	c.logger.ContextInfof(ctx, fmt.Sprint(operationInfo, " , duration: ", time.Now().Sub(beginTime).Milliseconds()))
	return
}
