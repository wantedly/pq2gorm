package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

//go:generate go-bindata _templates/

func main() {
	var (
		dir string
		ts  string
	)

	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	f.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: "+os.Args[0]+" <PostgreSQL URL> [<options>]\n\nOptions:\n")
		f.PrintDefaults() // Print usage of options
	}
	f.StringVar(&dir, "dir", "./", "Set output path")
	f.StringVar(&dir, "d", "./", "Set output path")
	f.StringVar(&ts, "tables", "", "Target tables (table1,table2,...) (default: all tables)")
	f.StringVar(&ts, "t", "", "Target tables (table1,table2,...) (default: all tables)")

	f.Parse(os.Args[1:])

	var url string

	for 0 < f.NArg() {
		url = f.Args()[0]
		f.Parse(f.Args()[1:])
	}

	if url == "" {
		f.Usage()
		os.Exit(1)
	}

	if err := os.MkdirAll(dir, 0777); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Connecting to database...\n")

	postgres, err := NewPostgres(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer postgres.DB.Close()

	var targets []string

	for _, t := range strings.Split(ts, ",") {
		if t != "" {
			targets = append(targets, t)
		}
	}

	tables, err := postgres.RetrieveTables(targets)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, table := range tables {
		fmt.Printf("Table name: %s\n", table)

		pkeys, err := postgres.RetrievePrimaryKeys(table)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fields, err := postgres.RetrieveFields(table)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if err := GenerateModel(table, pkeys, fields, dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
