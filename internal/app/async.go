package app

import (
	"context"
	"time"

	"github.com/atterpac/dado/async"
)

// RunAsync executes a function asynchronously with toast feedback.
func RunAsync[T any](
	message string,
	fn func(ctx context.Context) (T, error),
	onSuccess func(T),
	onError func(error),
) *async.Loader[T] {
	return async.NewLoader[T]().
		WithTimeout(30 * time.Second).
		WithIndicator(async.Toast(message)).
		OnSuccess(onSuccess).
		OnError(onError).
		Run(fn)
}

// RunAsyncSimple executes a void function asynchronously with toast feedback.
func RunAsyncSimple(
	message string,
	fn func(ctx context.Context) error,
	onSuccess func(),
	onError func(error),
) *async.Loader[struct{}] {
	return async.NewLoader[struct{}]().
		WithTimeout(30 * time.Second).
		WithIndicator(async.Toast(message)).
		OnSuccess(func(_ struct{}) {
			if onSuccess != nil {
				onSuccess()
			}
		}).
		OnError(onError).
		Run(func(ctx context.Context) (struct{}, error) {
			return struct{}{}, fn(ctx)
		})
}

// RunAsyncLong executes a long-running function with extended timeout.
func RunAsyncLong[T any](
	message string,
	timeout time.Duration,
	fn func(ctx context.Context) (T, error),
	onSuccess func(T),
	onError func(error),
) *async.Loader[T] {
	return async.NewLoader[T]().
		WithTimeout(timeout).
		WithIndicator(async.Toast(message)).
		OnSuccess(onSuccess).
		OnError(onError).
		Run(fn)
}
