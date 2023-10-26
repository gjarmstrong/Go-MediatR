package mediatr

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

var (
	requests = map[reflect.Type]any{}
	pipeline Middleware
)

type Request[T any] interface {
	Request(T)
}

type RequestHandler[R Request[T], T any] interface {
	Handle(context.Context, R) (T, error)
}

func RegisterRequestHandler[R Request[T], T any](rh RequestHandler[R, T]) error {
	var (
		req R
		rt  = reflect.TypeOf(req)
	)

	_, exists := requests[rt]
	if exists {
		return fmt.Errorf("handler for request %T already exists", req)
	}

	requests[rt] = rh

	return nil
}

type RequestHandlerFunc[R Request[T], T any] func(context.Context, R) (T, error)

func (f RequestHandlerFunc[R, T]) Handle(ctx context.Context, req R) (T, error) {
	return f(ctx, req)
}

func RegisterHandlerFunc[R Request[T], T any](f func(context.Context, R) (T, error)) error {
	return RegisterRequestHandler(RequestHandlerFunc[R, T](f))
}

type MiddlewareHandler interface {
	Handle(context.Context, any) (any, error)
}

type MiddlewareFunc func(context.Context, any) (any, error)

func (f MiddlewareFunc) Handle(ctx context.Context, req any) (any, error) {
	return f(ctx, req)
}

type Middleware func(MiddlewareFunc) MiddlewareFunc

func RegisterMiddleware(mw ...Middleware) {
	pipeline = func(outer Middleware, others ...Middleware) Middleware {
		return func(next MiddlewareFunc) MiddlewareFunc {
			for i := len(others) - 1; i >= 0; i-- { // reverse
				next = others[i](next)
			}

			if outer == nil {
				return next
			}

			return outer(next)
		}
	}(pipeline, mw...)
}

func Send[R Request[T], T any](ctx context.Context, req R) (T, error) {
	handler, ok := requests[reflect.TypeOf(req)]
	if !ok {
		return *new(T), fmt.Errorf("no handler for request %T", req)
	}

	rh, ok := handler.(RequestHandler[R, T])
	if !ok {
		return *new(T), fmt.Errorf("handler for request %T is not a Handler", req)
	}

	if pipeline == nil {
		return rh.Handle(ctx, req)
	}

	fn := func(ctx context.Context, req any) (res any, err error) {
		return rh.Handle(ctx, req.(R))
	}

	res, err := pipeline(fn)(ctx, req)
	if err != nil {
		return *new(T), err
	}

	return res.(T), nil
}

func ClearRequestRegistrations() {
	requests = map[reflect.Type]any{}
}

type NotificationHandlerFunc[T any] func(ctx context.Context, event T) error

var notifications = map[reflect.Type][]any{}

// RegisterNotificationHandler register the notification handler to mediatr registry.
func RegisterNotificationHandler[T any](handler NotificationHandlerFunc[T]) error {
	var event T
	eventType := reflect.TypeOf(event)

	handlers, exist := notifications[eventType]
	if !exist {
		notifications[eventType] = []any{handler}
		return nil
	}

	notifications[eventType] = append(handlers, handler)

	return nil
}

// RegisterNotificationHandlers register the notification handlers to mediatr registry.
func RegisterNotificationHandlers[T any](handlers ...NotificationHandlerFunc[T]) error {
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

// Publish the notification event to its corresponding notification handler.
func Publish[T any](ctx context.Context, event T) error {
	handlers, ok := notifications[reflect.TypeOf(event)]
	if !ok {
		return nil
	}

	for _, handler := range handlers {
		fn, ok := handler.(NotificationHandlerFunc[T])
		if !ok {
			return fmt.Errorf("handler for notification %T is not a Handler", event)
		}

		err := fn(ctx, event)
		if err != nil {
			return fmt.Errorf("error handling notification: %w", err)
		}
	}

	return nil
}

func ClearNotificationRegistrations() {
	notifications = map[reflect.Type][]any{}
}
