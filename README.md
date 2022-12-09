# go-batch-iterator
[![Actions Status](https://github.com/Eun/go-batch-iterator/workflows/CI/badge.svg)](https://github.com/Eun/go-batch-iterator/actions)
[![Coverage Status](https://coveralls.io/repos/github/Eun/go-batch-iterator/badge.svg?branch=master)](https://coveralls.io/github/Eun/go-batch-iterator?branch=master)
[![PkgGoDev](https://img.shields.io/badge/pkg.go.dev-reference-blue)](https://pkg.go.dev/github.com/Eun/go-batch-iterator)
[![GoDoc](https://godoc.org/github.com/Eun/go-batch-iterator?status.svg)](https://godoc.org/github.com/Eun/go-batch-iterator)
[![go-report](https://goreportcard.com/badge/github.com/Eun/go-batch-iterator)](https://goreportcard.com/report/github.com/Eun/go-batch-iterator)
[![go1.19](https://img.shields.io/badge/go-1.19-blue)](#)
---
go-batch-iterator provides an iterator to sequentially iterate over datasources utilizing a
batch approach.  
It utilizes generics to achieve an independence from underlying datastructures.
Therefore, it is ready to serve rows from a database or objects from an api endpoint with pagination.
All the user has to do is provide the logic to fetch the next batch from their datasource.

> go get -u github.com/Eun/go-batch-iterator

### Example Usage
```go
package batchiterator_test

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"testing"

	batchiterator "github.com/Eun/go-batch-iterator"
	_ "github.com/glebarez/go-sqlite"
)

func ExampleNewBatchIterator() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Print(err)
		return
	}
	defer db.Close()

	_, err = db.Exec(`
CREATE TABLE users (name TEXT);
INSERT INTO users (name) VALUES ("Alice"), ("Bob"), ("Charlie"), ("George"), ("Gerald"), ("Joe"), ("John"), ("Zoe");
`)
	if err != nil {
		log.Print(err)
		return
	}

	maxRowsPerQuery := 3
	offset := 0
	iter := batchiterator.Iterator[string]{
		NextBatchFunc: func(ctx context.Context) (items []string, hasMoreItems bool, err error) {
			rows, err := db.QueryContext(ctx, "SELECT name FROM users LIMIT ? OFFSET ?", maxRowsPerQuery, offset)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, false, nil
				}
				return nil, false, err
			}
			defer rows.Close()

			var names []string
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					return nil, false, err
				}
				names = append(names, name)
			}
			if err := rows.Err(); err != nil {
				return nil, false, err
			}
			offset += maxRowsPerQuery
			moreRowsAvailable := len(names) == maxRowsPerQuery
			return names, moreRowsAvailable, nil
		},
	}
	for iter.Next(context.Background()) {
		log.Print(*iter.Value())
	}
	if err := iter.Error(); err != nil {
		log.Print(err)
		return
	}
}

func TestExamples(t *testing.T) {
	ExampleNewBatchIterator()
}
```
[example_test.go](example_test.go)

### Options

**WithNextBatchFunc**  
Specify the next batch function, this function will be called by the `Next` function when the current batch
is depleted and the iterator needs to fetch the next batch.
This function should return the items for this batch (can be any type) and an indicator whether there are more
items expected.
Common uses are fetching rows from a database, or consuming a rest api with a next page token.
```go
batchiterator.NewBatchIterator(
    batchiterator.WithNextBatchFunc(func(ctx context.Context, iter batchiterator.Iterator[int]) ([]int, bool, error) {
		panic("not implemented")
    }),
)
```
```go
batchiterator.NewBatchIterator(
    batchiterator.WithNextBatchFunc(func(ctx context.Context, iter batchiterator.Iterator[int]) ([]string, bool, error) {
		panic("not implemented")
    }),
)
```
```go
type User struct {
	Name string
}
batchiterator.NewBatchIterator(
    batchiterator.WithNextBatchFunc(func(ctx context.Context, iter batchiterator.Iterator[int]) ([]User, bool, error) {
		panic("not implemented")
    }),
)
```

**WithStaticSlice**  
Sometimes a developer wants to use the iterator but the items are already fetched.
```go
batchiterator.NewBatchIterator(
    batchiterator.WithStaticSlice([]int{1, 2, 3}),
)
```
```go
batchiterator.NewBatchIterator(
    batchiterator.WithStaticSlice([]string{"A", "B", "C"}),
)
```
```go
type User struct {
    Name string
}
batchiterator.NewBatchIterator(
    batchiterator.WithStaticSlice([]User{{"Alice"}, {"Bob"}, {"Joe"}}),
)
```

**WithCloseFunc**  
Sometimes it is necessary to run custom logic after the iterator was closed, e.g. disconnecting database connection.  
```go
batchiterator.NewBatchIterator(
    batchiterator.WithNextBatchFunc(func(ctx context.Context, iter batchiterator.Iterator[int]) ([]User, bool, error) {
        panic("not implemented")
    }),
    batchiterator.WithCloseFunc(func(ctx context.Context, iter batchiterator.Iterator[string]) error {
        panic("not implemented")
    }),
)
```

**WithRateLimit**  
`WithRateLimit` takes a `rate.Limiter` and wraps the `NextBatchFunc` with this limit.
```go
batchiterator.NewBatchIterator(
    batchiterator.WithNextBatchFunc(func(ctx context.Context, iter batchiterator.Iterator[int]) ([]User, bool, error) {
        panic("not implemented")
    }),
    batchiterator.WithRateLimit[string](rate.NewLimiter(rate.Every(time.Second), 1)),
)
```
