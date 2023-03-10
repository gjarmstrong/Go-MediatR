package mediatr

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

type RequestHandlerFunc[TReq, TRes any] func(context.Context, TReq) (TRes, error)

func (f RequestHandlerFunc[TReq, TRes]) Handle(ctx context.Context, req any) (any, error) {
	return f(ctx, req.(TReq))
}

type PipelineFunc func(context.Context, any) (any, error)

func (f PipelineFunc) Handle(ctx context.Context, req any) (any, error) {
	return f(ctx, req)
}

type PipelineHandler interface {
	Handle(context.Context, any) (any, error)
}

type Pipeline func(PipelineFunc) PipelineFunc

var requests = map[reflect.Type]any{}
var pipeline Pipeline = func(rhf PipelineFunc) PipelineFunc { return rhf }

func RegisterHandlerFunc[TReq, TRes any](f RequestHandlerFunc[TReq, TRes]) error {
	var req TReq
	rt := reflect.TypeOf(req)
	_, exists := requests[rt]
	if exists {
		return errors.New("handler func already registered")
	}

	requests[rt] = f
	return nil
}

func RegisterPipeline(behaviors ...Pipeline) {
	pipeline = func(outer Pipeline, others ...Pipeline) Pipeline {
		return func(next PipelineFunc) PipelineFunc {
			for i := len(others) - 1; i >= 0; i-- { // reverse
				next = others[i](next)
			}
			return outer(next)
		}
	}(pipeline, behaviors...)
}

func Send[TReq, TRes any](ctx context.Context, req TReq) (TRes, error) {
	handler, ok := requests[reflect.TypeOf(req)]
	if !ok {
		return *new(TRes), fmt.Errorf("no handler for request %T", req)
	}

	_, ok = handler.(RequestHandlerFunc[TReq, TRes])
	if !ok {
		return *new(TRes), fmt.Errorf("handler for request %T is not a Handler", req)
	}

	ph, ok := handler.(PipelineHandler)
	if !ok {
		return *new(TRes), fmt.Errorf("handler for request %T is not a pipeline handler", req)
	}

	res, err := pipeline(ph.Handle)(ctx, req)
	if err != nil {
		return *new(TRes), err
	}

	return res.(TRes), nil
}

type NotificationHandler[TNotification any] interface {
	Handle(ctx context.Context, notification TNotification) error
}

var notificationHandlersRegistrations = map[reflect.Type][]any{}

// RegisterNotificationHandler register the notification handler to mediatr registry.
func RegisterNotificationHandler[TEvent any](handler NotificationHandler[TEvent]) error {
	var event TEvent
	eventType := reflect.TypeOf(event)

	handlers, exist := notificationHandlersRegistrations[eventType]
	if !exist {
		notificationHandlersRegistrations[eventType] = []any{handler}
		return nil
	}

	notificationHandlersRegistrations[eventType] = append(handlers, handler)

	return nil
}

// RegisterNotificationHandlers register the notification handlers to mediatr registry.
func RegisterNotificationHandlers[TEvent any](handlers ...NotificationHandler[TEvent]) error {
	if len(handlers) == 0 {
		return errors.New("no handlers provided")
	}

	for _, handler := range handlers {
		err := RegisterNotificationHandler(handler)
		if err != nil {
			return err
		}
	}

	return nil
}

func ClearRequestRegistrations() {
	requests = map[reflect.Type]any{}
}

func ClearNotificationRegistrations() {
	notificationHandlersRegistrations = map[reflect.Type][]any{}
}

// Publish the notification event to its corresponding notification handler.
func Publish[TNotification any](ctx context.Context, notification TNotification) error {
	eventType := reflect.TypeOf(notification)

	handlers, ok := notificationHandlersRegistrations[eventType]
	if !ok {
		// notification strategy should have zero or more handlers, so it should run without any error if we can't find a corresponding handler
		return nil
	}

	for _, handler := range handlers {
		handlerValue, ok := handler.(NotificationHandler[TNotification])
		if !ok {
			return fmt.Errorf("handler for notification %T is not a Handler", notification)
		}

		err := handlerValue.Handle(ctx, notification)
		if err != nil {
			return fmt.Errorf("error handling notification: %w", err)
		}
	}

	return nil
}
