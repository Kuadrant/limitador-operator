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
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	"go.opentelemetry.io/otel/trace"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// Tracer name
	tracerName = "limitador-operator"

	// Span names
	spanReconcile                    = "Reconcile"
	spanReconcileSpec                = "reconcileSpec"
	spanReconcileStatus              = "reconcileStatus"
	spanReconcileService             = "reconcileService"
	spanReconcilePVC                 = "reconcilePVC"
	spanReconcileDeployment          = "reconcileDeployment"
	spanReconcileLimitsConfigMap     = "reconcileLimitsConfigMap"
	spanReconcilePodDisruptionBudget = "reconcilePodDisruptionBudget"
	spanReconcilePodAnnotation       = "reconcilePodLimitsHashAnnotation"

	// Attribute keys for spans
	attrK8sLimitadorReplicas  = "k8s.limitador.replicas"
	attrK8sLimitadorStorage   = "k8s.limitador.storage.type"
	attrReconcileRequeue      = "reconcile.requeue"
	attrReconcileRequeueAfter = "reconcile.requeue_after"
	attrResourceOperation     = "k8s.resource.operation"

	// Resource operation values
	operationApplied   = "applied"
	operationCreated   = "created"
	operationUnchanged = "unchanged"

	// Event names (only for significant point-in-time occurrences)
	eventSpecCompleted   = "reconcile.spec.completed"
	eventStatusCompleted = "reconcile.status.completed"
)

// Tracer wraps the OpenTelemetry tracer with helper methods
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer creates a new Tracer
func NewTracer() *Tracer {
	return &Tracer{
		tracer: otel.Tracer(tracerName),
	}
}

// StartReconcileSpan starts a new span for the main reconcile function
// Accepts optional span start options (e.g., for adding links)
// Automatically refreshes the logger in context with the new span's trace context
func (t *Tracer) StartReconcileSpan(ctx context.Context, req ctrl.Request, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// Base options for reconciliation spans
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.K8SNamespaceNameKey.String(req.Namespace),
			attribute.String("k8s.limitador.name", req.Name),
		),
	}

	// Append any additional options (e.g., links)
	spanOpts = append(spanOpts, opts...)

	ctx, span := t.tracer.Start(ctx, spanReconcile, spanOpts...)

	// Refresh logger in context with new span's trace context
	ctx = RefreshLoggerWithCurrentSpan(ctx)

	return ctx, span
}

// StartReconcileSpecSpan starts a span for reconcileSpec
// Automatically refreshes the logger in context with the new span's trace context
func (t *Tracer) StartReconcileSpecSpan(ctx context.Context) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, spanReconcileSpec)

	// Refresh logger in context with new span's trace context
	ctx = RefreshLoggerWithCurrentSpan(ctx)

	return ctx, span
}

// StartReconcileStatusSpan starts a span for reconcileStatus
// Automatically refreshes the logger in context with the new span's trace context
func (t *Tracer) StartReconcileStatusSpan(ctx context.Context) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, spanReconcileStatus)

	// Refresh logger in context with new span's trace context
	ctx = RefreshLoggerWithCurrentSpan(ctx)

	return ctx, span
}

// StartResourceSpan starts a span for reconciling a specific resource
// Automatically refreshes the logger in context with the new span's trace context
func (t *Tracer) StartResourceSpan(ctx context.Context, resourceType, namespace, name string) (context.Context, trace.Span) {
	var spanName string
	switch resourceType {
	case "Service":
		spanName = spanReconcileService
	case "PersistentVolumeClaim":
		spanName = spanReconcilePVC
	case "Deployment":
		spanName = spanReconcileDeployment
	case "ConfigMap":
		spanName = spanReconcileLimitsConfigMap
	case "PodDisruptionBudget":
		spanName = spanReconcilePodDisruptionBudget
	case "PodAnnotation":
		spanName = spanReconcilePodAnnotation
	default:
		spanName = fmt.Sprintf("reconcile%s", resourceType)
	}

	ctx, span := t.tracer.Start(ctx, spanName,
		trace.WithAttributes(
			// Use semantic conventions where available
			semconv.K8SNamespaceNameKey.String(namespace),
			// K8s resource kind (e.g., "Deployment", "Service")
			attribute.String("k8s.resource.kind", resourceType),
			attribute.String("k8s.resource.name", name),
		),
	)

	// Refresh logger in context with new span's trace context
	ctx = RefreshLoggerWithCurrentSpan(ctx)

	return ctx, span
}

// RecordReconcileResult records the result of a reconciliation
func RecordReconcileResult(span trace.Span, result ctrl.Result, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.SetAttributes(
		attribute.String(attrReconcileRequeueAfter, result.RequeueAfter.String()),
	)
}

// SetResourceOperation sets the operation type as a span attribute.
// This follows OpenTelemetry best practices of using attributes rather than events
// for categorizing what happened during the span.
func SetResourceOperation(span trace.Span, operation string) {
	span.SetAttributes(attribute.String(attrResourceOperation, operation))
}

// RecordResourceApplied records that a resource was applied using server-side apply.
// With server-side apply, we don't distinguish between create/update/unchanged.
func RecordResourceApplied(span trace.Span) {
	SetResourceOperation(span, operationApplied)
}

// RecordResourceCreated records that a resource was created.
// Use this for create-only operations where we know creation occurred.
func RecordResourceCreated(span trace.Span) {
	SetResourceOperation(span, operationCreated)
}

// RecordResourceUnchanged records that a resource already exists and was not modified.
// Use this for operations that verified resource exists without changes.
func RecordResourceUnchanged(span trace.Span) {
	SetResourceOperation(span, operationUnchanged)
}

// RecordError records an error in the span
func RecordError(span trace.Span, err error, description string) {
	span.RecordError(err, trace.WithAttributes(
		attribute.String("error.description", description),
	))
	span.SetStatus(codes.Error, description)
}

// RecordSpecCompleted records that spec reconciliation completed
func RecordSpecCompleted(span trace.Span) {
	span.AddEvent(eventSpecCompleted)
	span.SetStatus(codes.Ok, "")
}

// RecordStatusCompleted records that status reconciliation completed
func RecordStatusCompleted(span trace.Span) {
	span.AddEvent(eventStatusCompleted)
	span.SetStatus(codes.Ok, "")
}

// AddLimitadorAttributes adds Limitador-specific attributes to a span
func AddLimitadorAttributes(span trace.Span, namespace, name string, replicas int32, storageType string) {
	span.SetAttributes(
		semconv.K8SNamespaceNameKey.String(namespace),
		attribute.String("k8s.limitador.name", name),
		attribute.Int(attrK8sLimitadorReplicas, int(replicas)),
		attribute.String(attrK8sLimitadorStorage, storageType),
	)
}
