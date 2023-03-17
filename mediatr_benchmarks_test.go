package mediatr

import (
	"context"
	"testing"
)

type RequestTest struct{ Data string }
type ResponseTest struct{ Data string }

func Benchmark_Send(b *testing.B) {
	ClearRequestRegistrations()

	fn := func() RequestHandlerFunc[RequestTest, ResponseTest] {
		return func(ctx context.Context, a RequestTest) (ResponseTest, error) { return ResponseTest(a), nil }
	}

	errRegister := RegisterHandlerFunc(fn())
	if errRegister != nil {
		b.Error(errRegister)
	}

	b.ResetTimer()
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		_, err := Send[RequestTest, ResponseTest](ctx, RequestTest{Data: "test"})
		if err != nil {
			b.Error(err)
		}
	}
}

type NotificationTest struct{ Data string }

func Benchmark_Publish(b *testing.B) {
	// because benchmark method will run multiple times, we need to reset the notification handlers registry before each run.
	ClearNotificationRegistrations()

	handler := func() NotificationHandlerFunc[NotificationTest] {
		return func(ctx context.Context, event NotificationTest) error { return nil }
	}

	handler2 := func() NotificationHandlerFunc[NotificationTest] {
		return func(ctx context.Context, event NotificationTest) error { return nil }
	}

	errRegister := RegisterNotificationHandlers(handler(), handler2())
	if errRegister != nil {
		b.Error(errRegister)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Publish(context.Background(), NotificationTest{Data: "test"})
		if err != nil {
			b.Error(err)
		}
	}
}
