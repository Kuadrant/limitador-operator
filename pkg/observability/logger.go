/*
Copyright 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package observability

import (
	"context"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
)

// baseLoggerKeyType is the type for the base logger context key
type baseLoggerKeyType struct{}

var baseLoggerKey = baseLoggerKeyType{}

// StoreBaseLogger stores the base logger (without trace context) in the context.
// This allows us to create fresh loggers with updated trace context as new spans are created.
func StoreBaseLogger(ctx context.Context, logger logr.Logger) context.Context {
	return context.WithValue(ctx, baseLoggerKey, logger)
}

// RefreshLoggerWithCurrentSpan updates the logger in context with the current span's trace_id and span_id.
// This should be called after creating a new span to ensure logs reflect the current span context.
func RefreshLoggerWithCurrentSpan(ctx context.Context) context.Context {
	// Get the base logger (without trace context)
	baseLogger, ok := ctx.Value(baseLoggerKey).(logr.Logger)
	if !ok {
		// No base logger stored, return context unchanged
		return ctx
	}

	// Get current span from context
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()

	if !spanCtx.IsValid() {
		// No valid span, use base logger without trace context
		return logr.NewContext(ctx, baseLogger)
	}

	// Create fresh logger with current span's trace context
	enrichedLogger := baseLogger.WithValues(
		"trace_id", spanCtx.TraceID().String(),
		"span_id", spanCtx.SpanID().String(),
	)

	return logr.NewContext(ctx, enrichedLogger)
}
