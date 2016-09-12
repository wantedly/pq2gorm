package main

import (
	"database/sql"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/gedex/inflector"
	_ "github.com/lib/pq"
)

func retrieveAllTables(db *sql.DB) (*sql.Rows, error) {
	return db.Query(`select relname as TABLE_NAME from pg_stat_user_tables`)
}

func retrieveTables(db *sql.DB, targets []string) (*sql.Rows, error) {
	qs := []string{}
	params := []interface{}{}

	for i, t := range targets {
		qs = append(qs, fmt.Sprintf("$%d", i+1))
		params = append(params, t)
	}

	return db.Query(fmt.Sprintf(`select relname as TABLE_NAME from pg_stat_user_tables where relname in (%s)`, strings.Join(qs, ", ")), params...)
}

func getTableNames(db *sql.DB, targets []string) ([]string, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if len(targets) == 0 {
		rows, err = retrieveAllTables(db)
		if err != nil {
			return nil, err
		}
	} else {
		rows, err = retrieveTables(db, targets)
		if err != nil {
			return nil, err
		}
	}

	tableNames := []string{}
	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			return nil, err
		}

		tableNames = append(tableNames, tableName)
	}

	return tableNames, nil
}

func genModel(tableName string, outPath string, db *sql.DB) error {
	primaryKeys, err := getPrimaryKeys(tableName, db)
	if err != nil {
		return err
	}

	query :=
		`
    select column_name, data_type, COALESCE(column_default, '') as column_default, is_nullable
    from information_schema.columns
    where
      table_name='` + tableName + `'
    order by
      ordinal_position;
    `

	rows, err := db.Query(query)
	if err != nil {
		return err
	}

	var gormStr string
	var needTimePackage bool
	for rows.Next() {
		var (
			columnName    string
			dataType      string
			columnDefault string
			isNullable    string
		)

		err = rows.Scan(&columnName, &dataType, &columnDefault, &isNullable)
		if err != nil {
			return err
		}

		json := genJSON(columnName, columnDefault, primaryKeys)
		fieldType := gormDataType(dataType)

		if fieldType == "time.Time" || fieldType == "*time.Time" {
			needTimePackage = true

			if isNullable == "YES" {
				fieldType = "*time.Time"
			} else {
				fieldType = "time.Time"
			}
		}

		if fieldType == "double precision" {
			fieldType = "float32"
		}

		m := gormColName(columnName) + " " + fieldType + " `" + json + "`\n"
		gormStr += m

		isInfered, infColName := inferORM(columnName)

		// Add belongs_to relation
		if isInfered {
			json := genJSON(strings.ToLower(infColName), "", nil)
			comment := "// This line is infered from column name \"" + columnName + "\"."
			infColName = gormColName(infColName)

			m := infColName + " *" + infColName + " `" + json + "` " + comment + "\n"
			gormStr += m
		}
	}

	var importPackage string
	if needTimePackage {
		importPackage = "import \"time\"\n\n"
	} else {
		importPackage = ""
	}

	gormStr = "package models\n\n" + importPackage + "type " + gormTableName(tableName) + " struct {\n" + gormStr + "}\n"

	modelFile := filepath.Join(outPath, inflector.Singularize(tableName)+".go")
	file, err := os.Create(modelFile)

	if err != nil {
		return err
	}

	defer file.Close()

	src, err := format.Source(([]byte)(gormStr))
	if err != nil {
		return err
	}

	file.Write(src)

	return nil
}

func getPrimaryKeys(tableName string, db *sql.DB) (map[string]bool, error) {
	query :=
		`
    select
    ccu.column_name as COLUMN_NAME
    from
      information_schema.table_constraints tc
      ,information_schema.constraint_column_usage ccu
    where
      tc.table_name='` + tableName + `'
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

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	primaryKeys := map[string]bool{}
	for rows.Next() {
		var columnName string
		err = rows.Scan(&columnName)
		if err != nil {
			return nil, err
		}

		primaryKeys[columnName] = true
	}

	return primaryKeys, nil
}
