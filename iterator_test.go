package batchiterator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	batchiterator "github.com/Eun/go-batch-iterator"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func getAllValuesFromIter[T any](iter batchiterator.Iterator[T]) ([]T, error) {
	var items []T
	for iter.Next(context.Background()) {
		items = append(items, *iter.Value())
	}
	return items, iter.Error()
}

func FuzzWithStaticSlice(f *testing.F) {
	testcases := []string{
		"",
		"A",
		"AB",
		"ABC",
	}
	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, s string) {
		v := []byte(s)
		if len(v) == 0 {
			v = nil
		}
		iter := batchiterator.Iterator[byte]{
			NextBatchFunc: batchiterator.StaticSlice([]byte(s)),
		}
		values, err := getAllValuesFromIter(iter)
		require.NoError(t, err)

		require.Equal(t, v, values)
	})
}

func TestWithNextBatchFunc(t *testing.T) {
	t.Parallel()

	var idx int

	iter := batchiterator.Iterator[string]{
		NextBatchFunc: func(context.Context) ([]string, bool, error) {
			idx++
			switch idx {
			case 1:
				return []string{"A", "B", "C"}, true, nil
			case 2:
				return []string{"D"}, true, nil
			case 3:
				return []string{"E", "F"}, true, nil
			default:
				return nil, false, nil
			}
		},
	}
	values, err := getAllValuesFromIter(iter)
	require.NoError(t, err)

	require.Equal(t, []string{"A", "B", "C", "D", "E", "F"}, values)
}

func TestWithNextBatchFuncAndEmptyResponse(t *testing.T) {
	t.Parallel()

	var idx int

	iter := batchiterator.Iterator[string]{
		NextBatchFunc: func(context.Context) ([]string, bool, error) {
			idx++
			switch idx {
			case 1:
				return []string{"A", "B", "C"}, true, nil
			case 2:
				return nil, true, nil
			case 3:
				return []string{"D", "E", "F"}, true, nil
			default:
				return nil, false, nil
			}
		},
	}
	values, err := getAllValuesFromIter(iter)
	require.NoError(t, err)

	require.Equal(t, []string{"A", "B", "C", "D", "E", "F"}, values)
}

func FuzzOverflow(f *testing.F) {
	iter := batchiterator.Iterator[string]{
		NextBatchFunc: batchiterator.StaticSlice([]string{"A", "B", "C"}),
	}

	for iter.Next(context.Background()) {
	}
	require.NoError(f, iter.Error())

	testcases := []int{3, 6, 9}
	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, n int) {
		for i := 0; i < n; i++ {
			require.Nil(t, iter.Value())
			require.False(t, iter.Next(context.Background()))
		}
	})
}

func TestErrorDuringNextBatchFunc(t *testing.T) {
	t.Parallel()

	t.Run("with no more data", func(t *testing.T) {
		t.Parallel()

		iter := batchiterator.Iterator[string]{
			NextBatchFunc: func(context.Context) ([]string, bool, error) {
				return nil, false, errors.New("some error")
			},
		}

		require.False(t, iter.Next(context.Background()))
		require.EqualError(t, iter.Error(), "some error")
	})

	t.Run("with more data", func(t *testing.T) {
		t.Parallel()

		iter := batchiterator.Iterator[string]{
			NextBatchFunc: func(context.Context) ([]string, bool, error) {
				return nil, true, errors.New("some error")
			},
		}

		require.False(t, iter.Next(context.Background()))
		require.EqualError(t, iter.Error(), "some error")
	})
}

func TestMissingNextBatchFunc(t *testing.T) {
	t.Parallel()

	iter := batchiterator.Iterator[string]{}
	require.False(t, iter.Next(context.Background()))
	require.Nil(t, iter.Value())
	require.EqualError(t, iter.Error(), "iterator has no next batch function")
}

func TestNextBatchFunctionChangedInFlight(t *testing.T) {
	t.Parallel()

	iter := batchiterator.Iterator[string]{
		NextBatchFunc: batchiterator.StaticSlice([]string{"A", "B", "C"}),
	}
	require.True(t, iter.Next(context.Background()))
	require.Equal(t, "A", *iter.Value())
	iter.NextBatchFunc = nil
	require.False(t, iter.Next(context.Background()))
	require.Nil(t, iter.Value())
	require.EqualError(t, iter.Error(), "iterator has no next batch function")
}

func TestWithRateLimit(t *testing.T) {
	t.Parallel()

	t.Run("one request per second", func(t *testing.T) {
		var idx int
		var lastCall time.Time

		iter := batchiterator.Iterator[string]{
			NextBatchFunc: batchiterator.RateLimit(
				rate.NewLimiter(rate.Every(time.Second), 1),
				func(context.Context) ([]string, bool, error) {
					require.Greater(t, time.Since(lastCall), time.Second-time.Millisecond*100)
					lastCall = time.Now()
					idx++
					switch idx {
					case 1:
						return []string{"A", "B", "C"}, true, nil
					case 2:
						return nil, true, nil
					case 3:
						return []string{"D", "E", "F"}, true, nil
					default:
						return nil, false, nil
					}
				},
			),
		}
		values, err := getAllValuesFromIter(iter)
		require.NoError(t, err)

		require.Equal(t, []string{"A", "B", "C", "D", "E", "F"}, values)
	})

	t.Run("nil limiter", func(t *testing.T) {
		iter := batchiterator.Iterator[string]{
			NextBatchFunc: batchiterator.RateLimit(
				nil,
				func(context.Context) ([]string, bool, error) {
					return nil, false, nil
				},
			),
		}
		require.False(t, iter.Next(context.Background()))
		require.Nil(t, iter.Value())
		require.EqualError(t, iter.Error(), "limiter cannot be nil")
	})
}
