package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/wantedly/pq2gorm/pq2g"
	_ "github.com/lib/pq"
)

func main() {
	var dir string

	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	f.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: Generate gorm model structs from PostgreSQL database schema.\n")
		f.PrintDefaults() // Print usage of options
	}
	f.StringVar(&dir, "dir", "./", "Set output path")
	f.StringVar(&dir, "d", "./", "Set output path")

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

	db, err := sql.Open("postgres", pgURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()

	tables, err := pq2g.GetTableName(db)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, tableName := range tables {
		fmt.Printf("Table name: %s\n", tableName)

		if err := pq2g.GenModel(tableName, dir, db); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
