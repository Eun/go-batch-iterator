package batchiterator_test

import (
	"strconv"
	"testing"
)

func TestExamples(t *testing.T) {
	tests := []func(){
		ExampleIterator_sql,
		ExampleIterator_pagination,
	}

	for i, tt := range tests {
		tt := tt
		t.Run(strconv.Itoa(i), func(*testing.T) {
			tt()
		})
	}
}
