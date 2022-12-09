# go-batch-iterator
[![Actions Status](https://github.com/Eun/go-batch-iterator/workflows/push/badge.svg)](https://github.com/Eun/go-batch-iterator/actions)
[![Coverage Status](https://coveralls.io/repos/github/Eun/go-batch-iterator/badge.svg?branch=master)](https://coveralls.io/github/Eun/go-batch-iterator?branch=master)
[![PkgGoDev](https://img.shields.io/badge/pkg.go.dev-reference-blue)](https://pkg.go.dev/github.com/Eun/go-batch-iterator)
[![GoDoc](https://godoc.org/github.com/Eun/go-batch-iterator?status.svg)](https://godoc.org/github.com/Eun/go-batch-iterator)
[![go-report](https://goreportcard.com/badge/github.com/Eun/go-batch-iterator)](https://goreportcard.com/report/github.com/Eun/go-batch-iterator)
[![go1.18](https://img.shields.io/badge/go-1.18-blue)](#)
---
*go-batch-iterator* provides an iterator to sequentially iterate over datasources utilizing a
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

func ExampleIterator() {
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
	ExampleIterator()
}
```
[example_test.go](example_test.go)

### Helpers
[**StaticSlice**](option.go:#L11)  
Sometimes a developer wants to use the iterator but the items are already fetched.
```go
iter := batchiterator.Iterator[string]{
    NextBatchFunc: batchiterator.StaticSlice([]string{"A", "B", "C"}),
}
```


[**RateLimit**](option.go:#L18)  
`RateLimit` takes a `rate.Limiter` and wraps the `NextBatchFunc` with this limit.
```go
iter := batchiterator.Iterator[string]{
    NextBatchFunc: batchiterator.RateLimit(
		rate.NewLimiter(rate.Every(time.Second), 1), 
		func (ctx context.Context) (items []string, hasMoreItems bool, err error) {
			panic("not implemented")
		}, 
	),
}
```
