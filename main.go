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
		fmt.Fprintf(os.Stderr, `Usage of %s:
  %s <PostgreSQL URL> [<options>]

Options:
`, os.Args[0], os.Args[0])
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

	fmt.Println("Connecting to database...")

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

	paramsS := []*TemplateParams{}

	for _, table := range tables {
		fmt.Println("Table name: " + table)

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

		paramsS = append(paramsS, GenerateModel(table, pkeys, fields, tables))
	}

	for i, table := range tables {
		fmt.Println("Add relation for Table name: " + table)

		AddHasMany(paramsS[i])
		SaveModel(table, paramsS[i], dir)
	}
}
