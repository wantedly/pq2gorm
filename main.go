package main

import (
	"database/sql"
	"fmt"
	"net"
	nurl "net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/gedex/inflector"
	_ "github.com/lib/pq"
)

var db *sql.DB
var outDir = "./"

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: error: %s", err.Error())
		os.Exit(1)
	}
}

func getTableName() (tableNames []string) {
	query := `select relname as TABLE_NAME from pg_stat_user_tables`

	rows, err := db.Query(query)
	checkError(err)

	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		checkError(err)

		tableNames = append(tableNames, tableName)
	}

	return
}

func getPrimaryKeys(tableName string) (primaryKeys map[string]bool) {
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
	checkError(err)

	primaryKeys = map[string]bool{}
	for rows.Next() {
		var columnName string
		err = rows.Scan(&columnName)
		checkError(err)

		primaryKeys[columnName] = true
	}

	return
}

func genModel(tableNames []string) {
	for _, tableName := range tableNames {

		primaryKeys := getPrimaryKeys(tableName)

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
		checkError(err)

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
			checkError(err)

			json := genj(columnName, columnDefault, primaryKeys)

			if dataType == "timestamp with time zone" {
				needTimePackage = true
			}

			// If have to use pointer
			if dataType == "timestamp with time zone" && isNullable == "YES" {
				if hasNullRecoreds(tableName, columnName) == true {
					dataType = "*time.Time"
				}
			}

			m := gormColName(columnName) + " " + gormDataType(dataType) + " `" + json + "`\n"
			gormStr += m

			isInfered, infColName := inferORM(columnName)

			// Add belongs_to relation
			if isInfered == true {
				json := genj(strings.ToLower(infColName), "", nil)
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

		// fmt.Println(gormStr) // Print output

		file, err := os.Create(outDir + inflector.Singularize(tableName) + `.go`)
		checkError(err)
		defer file.Close()
		file.Write(([]byte)(gormStr))
	}

	err := exec.Command("gofmt", "-w", outDir).Run()
	checkError(err)
}

// Infer belongs_to Relation from column's name
func inferORM(s string) (bool, string) {
	s = strings.ToLower(s)
	ss := strings.Split(s, "_")

	const (
		id = "id"
	)

	newSS := []string{}
	var containsID bool = false
	for _, word := range ss {
		if word == id {
			containsID = true
			continue
		}

		newSS = append(newSS, word)
	}

	if containsID == false || len(newSS) == 0 {
		return false, ""
	}

	infColName := strings.Join(newSS, "_")
	return true, infColName
}

// Generate json
func genj(columnName, columnDefault string, primaryKeys map[string]bool) (json string) {
	json = "json:\"" + columnName + "\""

	if primaryKeys[columnName] == true {
		p := "gorm:\"primary_key;AUTO_INCREMENT\" "
		json = p + json
	}

	if columnDefault != "" && !strings.Contains(columnDefault, "nextval") {
		d := " sql:\"DEFAULT:" + columnDefault + "\""
		json += d
	}

	return
}

func hasNullRecoreds(tableName string, columnName string) bool {
	query := `SELECT COUNT(*) FROM ` + tableName + ` WHERE ` + columnName + ` IS NULL;`

	var count string

	err := db.QueryRow(query).Scan(&count)
	checkError(err)

	val, _ := strconv.ParseInt(count, 10, 64)

	if val > 0 {
		return true
	}

	return false
}

// Singlarlize table name and upper initial character
func gormTableName(s string) string {
	s = strings.ToLower(s)
	s = inflector.Singularize(s)
	return strings.Title(s)
}

// Ex: facebook_uid → FacebookUID
func gormColName(s string) string {
	s = strings.ToLower(s)
	ss := strings.Split(s, "_")
	const (
		id  = "id"
		url = "url"
	)
	for i, word := range ss {
		if strings.Contains(word, id) {
			word = strings.Replace(word, id, "ID", -1)
		}
		if strings.Contains(word, url) {
			word = strings.Replace(word, url, "URL", -1)
		}
		ss[i] = strings.Title(word)
	}
	return strings.Join(ss, "")
}

func gormDataType(s string) string {
	switch s {
	case "integer":
		return "uint"
	case "numeric":
		return "float64"
	case "character varying", "text":
		return "string"
	case "boolean":
		return "bool"
	case "timestamp with time zone":
		return "time.Time"
	default:
		return s
	}
}

// not used
func parseURL(url string) (map[string]string, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, fmt.Errorf("invalid connection protocol: %s", u.Scheme)
	}

	kv := map[string]string{}
	escaper := strings.NewReplacer(` `, `\ `, `'`, `\'`, `\`, `\\`)
	accrue := func(k, v string) {
		if v != "" {
			kv[k] = escaper.Replace(v)
		}
	}

	if u.User != nil {
		v := u.User.Username()
		accrue("user", v)

		v, _ = u.User.Password()
		accrue("password", v)
	}

	if host, port, err := net.SplitHostPort(u.Host); err != nil {
		accrue("host", u.Host)
	} else {
		accrue("host", host)
		accrue("port", port)
	}

	if u.Path != "" {
		accrue("dbname", u.Path[1:])
	}

	q := u.Query()
	for k := range q {
		accrue(k, q.Get(k))
	}

	return kv, nil
}

func main() {
	app := cli.NewApp()
	app.Name = "pq2gorm"
	app.Usage = "Generate gorm model structs from PostgreSQL database schema"
	app.Version = "0.0.1"

	// global options
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "dry-run, d",
			Usage: "dry-run",
		},
	}

	app.Action = func(c *cli.Context) error {

		var paramFirst = ""
		if len(c.Args()) > 0 {

			if len(c.Args()) == 2 {
				outDir = c.Args()[1] + "/"
			}

			if len(c.Args()) > 2 {
				fmt.Println("Too many arguments are given")
				return nil
			}

			var isDry = c.GlobalBool("dry-run")

			if isDry {
				fmt.Println("this is dry-run")
			} else {
				paramFirst = c.Args()[0]

				fmt.Printf("Connecting \"%s\"...\n", paramFirst)

				var err error
				db, err = sql.Open("postgres", paramFirst)
				checkError(err)
				defer db.Close()
				tables := getTableName()

				fmt.Println("Generating gorm from tables below...")
				for _, tableName := range tables {
					fmt.Printf("Table name: %s\n", tableName)
				}

				genModel(tables)
			}
		}

		return nil
	}

	app.Before = func(c *cli.Context) error {
		return nil
	}

	app.After = func(c *cli.Context) error {
		return nil
	}

	app.Run(os.Args)
}
