package batchiterator

import (
	"context"
	"errors"

	"golang.org/x/time/rate"
)

// StaticSlice sets the next batch function to use a static slice.
func StaticSlice[T any](s []T) NextBatchFunction[T] {
	return func(ctx context.Context) (items []T, hasMoreItems bool, err error) {
		return s, false, nil
	}
}

// RateLimit wraps the NextBatchFunc with a rate.Limiter.
func RateLimit[T any](limiter *rate.Limiter, nextBatchFunction NextBatchFunction[T]) NextBatchFunction[T] {
	return func(ctx context.Context) (items []T, hasMoreItems bool, err error) {
		if limiter == nil {
			return nil, false, errors.New("limiter cannot be nil")
		}
		if err := limiter.Wait(ctx); err != nil {
			return nil, false, err
		}
		return nextBatchFunction(ctx)
	}
}
