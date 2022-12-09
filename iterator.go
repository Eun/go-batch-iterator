// Package batchiterator implements an Iterator that fetches batches of items
// in the background transparently from the user.
package batchiterator

import (
	"context"
	"errors"
	"io"
)

// NextBatchFunction is the function that will be called by the Iterator.Next function to fetch the next batch.
// It returns the items for the batch and indicates whether there are more items available or not.
// If it is unclear whether there are more items or not, it should return true and the Iterator will call the function
// again. In this case it can be useful to delay the execution by using the rate limit helper (RateLimit).
// If all items are depleted it returns the items (or nil) and false for hasMoreItems.
// If an error occurred the iterator will stop the execution.
type NextBatchFunction[T any] func(ctx context.Context) (items []T, hasMoreItems bool, err error)

// Iterator is the main object for the iterator.
type Iterator[T any] struct {
	// NextBatchFunc is the function that will be called by the iterator to fetch the next batch.
	// For more details see NextBatchFunction.
	NextBatchFunc NextBatchFunction[T]
	batch         []T
	lastError     error
	isLastBatch   bool
}

// Next prepares the next item for reading with the Value method. It
// returns true on success, or false if there is no next value or an error
// happened while preparing it. Error should be consulted to distinguish between
// the two cases.
//
// It calls the NextBatchFunc after all items in the batch have been depleted.
func (iter *Iterator[T]) Next(ctx context.Context) bool {
	if iter.NextBatchFunc == nil {
		iter.batch = nil
		iter.lastError = errors.New("iterator has no next batch function")
		return false
	}
	if iter.lastError != nil {
		iter.batch = nil
		return false
	}
	if len(iter.batch) > 1 {
		iter.batch = iter.batch[1:]
		return true
	}
	if iter.isLastBatch {
		iter.batch = nil
		return false
	}
	var hasMoreItems bool
	iter.batch, hasMoreItems, iter.lastError = iter.NextBatchFunc(ctx)
	iter.isLastBatch = !hasMoreItems
	if iter.lastError != nil {
		iter.batch = nil
		return false
	}
	if len(iter.batch) == 0 {
		if !iter.isLastBatch {
			return iter.Next(ctx)
		}
		iter.batch = nil
		iter.lastError = io.EOF
		return false
	}
	return true
}

// Value returns the current value as a pointer in the batch.
func (iter *Iterator[T]) Value() *T {
	if len(iter.batch) == 0 {
		return nil
	}
	return &iter.batch[0]
}

// Error returns the current error.
func (iter *Iterator[T]) Error() error {
	if iter.lastError == io.EOF {
		return nil
	}
	return iter.lastError
}
