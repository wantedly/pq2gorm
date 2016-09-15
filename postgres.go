package main

import (
	"database/sql"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

type Postgres struct {
	DB *sql.DB
}

type Field struct {
	Name     string
	Type     string
	Default  string
	Nullable bool
}

func NewPostgres(url string) (*Postgres, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	return &Postgres{
		DB: db,
	}, nil
}

func (p *Postgres) retrieveAllTables() (*sql.Rows, error) {
	return p.DB.Query(`select relname as TABLE_NAME from pg_stat_user_tables`)
}

func (p *Postgres) retrieveSelectedTables(targets []string) (*sql.Rows, error) {
	qs := []string{}
	params := []interface{}{}

	for i, t := range targets {
		qs = append(qs, "$"+strconv.Itoa(i+1))
		params = append(params, t)
	}

	return p.DB.Query(`select relname as TABLE_NAME from pg_stat_user_tables where relname in (`+strings.Join(qs, ", ")+`)`, params...)
}

func (p *Postgres) RetrieveFields(table string) ([]*Field, error) {
	query :=
		`
    select column_name, data_type, COALESCE(column_default, '') as column_default, is_nullable
    from information_schema.columns
    where
      table_name='` + table + `'
    order by
      ordinal_position;
    `

	rows, err := p.DB.Query(query)
	if err != nil {
		return nil, err
	}

	var (
		columnName       string
		columnType       string
		columnDefault    string
		columnIsNullable string
	)

	var nullable bool

	fields := []*Field{}

	for rows.Next() {
		err = rows.Scan(&columnName, &columnType, &columnDefault, &columnIsNullable)
		if err != nil {
			return nil, err
		}

		if columnIsNullable == "YES" {
			nullable = true
		} else {
			nullable = false
		}

		field := &Field{
			Name:     columnName,
			Type:     columnType,
			Default:  columnDefault,
			Nullable: nullable,
		}
		fields = append(fields, field)
	}

	return fields, nil
}

func (p *Postgres) RetrieveTables(targets []string) ([]string, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if len(targets) == 0 {
		rows, err = p.retrieveAllTables()
		if err != nil {
			return nil, err
		}
	} else {
		rows, err = p.retrieveSelectedTables(targets)
		if err != nil {
			return nil, err
		}
	}

	tables := []string{}
	var table string

	for rows.Next() {
		err = rows.Scan(&table)
		if err != nil {
			return nil, err
		}

		tables = append(tables, table)
	}

	return tables, nil
}

func (p *Postgres) RetrievePrimaryKeys(table string) (map[string]bool, error) {
	query :=
		`
    select
    ccu.column_name as COLUMN_NAME
    from
      information_schema.table_constraints tc
      ,information_schema.constraint_column_usage ccu
    where
      tc.table_name='` + table + `'
      and
      tc.constraint_type='PRIMARY KEY'
      and
      tc.table_catalog=ccu.table_catalog
      and
      tc.table_schema=ccu.table_schema
      and
      tc.table_name=ccu.table_name
      and
      tc.constraint_name=ccu.constraint_name
    `

	rows, err := p.DB.Query(query)
	if err != nil {
		return nil, err
	}

	var column string
	pkeys := map[string]bool{}

	for rows.Next() {
		err = rows.Scan(&column)
		if err != nil {
			return nil, err
		}

		pkeys[column] = true
	}

	return pkeys, nil
}
