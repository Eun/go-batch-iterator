package batchiterator_test

import (
	"context"
	"database/sql"
	"errors"
	"log"

	batchiterator "github.com/Eun/go-batch-iterator"
	_ "github.com/glebarez/go-sqlite"
)

func database() *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Panicln(err)
	}

	_, err = db.Exec(`
CREATE TABLE users (name TEXT);
INSERT INTO users (name) VALUES ("Alice"), ("Bob"), ("Charlie"), ("George"), ("Gerald"), ("Joe"), ("John"), ("Zoe");
`)
	if err != nil {
		log.Panicln(err)
	}
	return db
}

func getNextUsersFromDB(db *sql.DB, offset *int, maxRowsPerQuery int) func(ctx context.Context) (items []string, hasMoreItems bool, err error) {
	return func(ctx context.Context) (items []string, hasMoreItems bool, err error) {
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
		*offset += maxRowsPerQuery
		moreRowsAvailable := len(names) == maxRowsPerQuery
		return names, moreRowsAvailable, nil
	}
}

func ExampleIterator_sql() {
	db := database()
	defer db.Close()

	maxRowsPerQuery := 3
	offset := 0
	iter := batchiterator.Iterator[string]{
		NextBatchFunc: getNextUsersFromDB(db, &offset, maxRowsPerQuery),
	}
	for iter.Next(context.Background()) {
		log.Print(*iter.Value())
	}
	if err := iter.Error(); err != nil {
		log.Print(err)
		return
	}
}
