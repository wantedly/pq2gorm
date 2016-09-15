package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

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

	var pgURL string

	for 0 < f.NArg() {
		pgURL = f.Args()[0]
		f.Parse(f.Args()[1:])
	}

	if pgURL == "" {
		f.Usage()
		os.Exit(1)
	}

	if err := os.MkdirAll(dir, 0777); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Connecting to database...\n")

	postgres, err := NewPostgres(pgURL)
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

	tables, err := postgres.RetrieveTableNames(targets)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, tableName := range tables {
		fmt.Printf("Table name: %s\n", tableName)

		if err := postgres.GenModel(tableName, dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
