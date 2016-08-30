package pq2g

import (
	"database/sql"
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gedex/inflector"
	_ "github.com/lib/pq"
)

func GetTableName(db *sql.DB) ([]string, error) {
	query := `select relname as TABLE_NAME from pg_stat_user_tables`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
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

func GenModel(tableName string, outPath string, db *sql.DB) error {
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

		if dataType == "timestamp with time zone" {
			needTimePackage = true
		}

		// If have to use pointer
		if dataType == "timestamp with time zone" && isNullable == "YES" {
			hasNullRecords, err := hasNullRecords(tableName, columnName, db)
			if err != nil {
				return err
			}

			if hasNullRecords {
				dataType = "*time.Time"
			}
		}

		m := gormColName(columnName) + " " + gormDataType(dataType) + " `" + json + "`\n"
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

func hasNullRecords(tableName string, columnName string, db *sql.DB) (bool, error) {
	query := `SELECT COUNT(*) FROM ` + tableName + ` WHERE ` + columnName + ` IS NULL;`

	var count string

	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return false, err
	}

	val, _ := strconv.ParseInt(count, 10, 64)

	if val > 0 {
		return true, nil
	}

	return false, nil
}
