package main

import (
	"testing"
)

func TestGormTableName(t *testing.T) {
	var testdata = []struct {
		in  string
		out string
	}{
		{"events", "Event"},
		{"post_comments", "PostComment"},
	}

	for _, td := range testdata {
		s := gormTableName(td.in)

		if s != td.out {
			t.Fatalf("Table name does not match. expect: %s, actual: %s", td.out, s)
		}
	}
}
