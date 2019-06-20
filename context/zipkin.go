package context

import (
	"context"
	"github.com/openzipkin/zipkin-go/model"
)

func WithSpanContext(ctx context.Context, c model.SpanContext) context.Context {
	return context.WithValue(ctx, spanParentKey, c)
}

func GetSpanContext(ctx context.Context) model.SpanContext {
	return ctx.Value(spanParentKey).(model.SpanContext)
}

type contextKey int

const spanParentKey contextKey = iota
