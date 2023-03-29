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
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	batchiterator "github.com/Eun/go-batch-iterator"
	"golang.org/x/time/rate"
)


func GetNextUsers(nextPageToken *string) func(ctx context.Context) (items []string, hasMoreItems bool, err error) {
	return func(ctx context.Context) (items []string, hasMoreItems bool, err error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8080/users", http.NoBody)
		if err != nil {
			return nil, false, err
		}
		req.Header.Set("X-NextPageToken", *nextPageToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, false, err
		}
		defer resp.Body.Close()
		
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			// too many requests retry later
			return nil, true, err
		case http.StatusNoContent:
			// no more items
			return nil, false, nil
		case http.StatusOK:
			var users []string
			*nextPageToken = resp.Header.Get("X-NextPageToken")
			err := json.NewDecoder(resp.Body).Decode(&users)
			return users, true, err
		default:
			return nil, false, errors.New("unknown status code")
		}
	}
}
func ExampleIterator_Pagination() {
	var nextPageToken string
	iter := batchiterator.Iterator[string]{
		NextBatchFunc: batchiterator.RateLimit(
			rate.NewLimiter(rate.Every(time.Second), 1),
			GetNextUsers(&nextPageToken),
		),
	}
	for iter.Next(context.Background()) {
		log.Print(*iter.Value())
	}
	if err := iter.Error(); err != nil {
		log.Print(err)
		return
	}
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
