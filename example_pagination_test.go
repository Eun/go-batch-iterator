package batchiterator_test

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	batchiterator "github.com/Eun/go-batch-iterator"
	_ "github.com/glebarez/go-sqlite"
	"golang.org/x/time/rate"
)

func httpServer() *http.Server {
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-NextPageToken") {
		case "":
			w.Header().Set("X-NextPageToken", "A")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]string{"Alice", "Bob"})
			return
		case "A":
			w.Header().Set("X-NextPageToken", "B")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]string{"Charlie"})
			return
		default:
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})
	go func() {
		if err := server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Panicln(err)
			}
		}
	}()
	return &server
}

func getNextUsersFromAPI(nextPageToken *string) func(ctx context.Context) (items []string, hasMoreItems bool, err error) {
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

func ExampleIterator_pagination() {
	server := httpServer()
	defer server.Shutdown(context.Background())

	var nextPageToken string
	iter := batchiterator.Iterator[string]{
		NextBatchFunc: batchiterator.RateLimit(
			rate.NewLimiter(rate.Every(time.Second), 1),
			getNextUsersFromAPI(&nextPageToken),
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
