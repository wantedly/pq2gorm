package main

import (
	"database/sql"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/gedex/inflector"
	_ "github.com/lib/pq"
	"net"
	nurl "net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var DB *sql.DB
var OutDir = "./"

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: error: %s", err.Error())
		os.Exit(1)
	}
}

func getTableName() (t_names []string) {
	query := `select relname as TABLE_NAME from pg_stat_user_tables`

	rows, err := DB.Query(query)
	checkError(err)

	for rows.Next() {
		var t_name string
		err = rows.Scan(&t_name)
		checkError(err)

		t_names = append(t_names, t_name)
	}

	return
}

func getPrimaryKey(t_name string) (c_names map[string]bool) {
	query :=
		`
    select
    ccu.column_name as COLUMN_NAME
    from
      information_schema.table_constraints tc
      ,information_schema.constraint_column_usage ccu
    where
      tc.table_name='` + t_name + `'
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

	rows, err := DB.Query(query)
	checkError(err)

	c_names = map[string]bool{}
	for rows.Next() {
		var c_name string
		err = rows.Scan(&c_name)
		checkError(err)

		c_names[c_name] = true
	}

	return
}

func genModel(t_names []string) {
	for _, t_name := range t_names {

		primary_key := getPrimaryKey(t_name)

		query :=
			`
      select column_name, data_type, COALESCE(column_default, '') as column_default, is_nullable
      from information_schema.columns
      where
        table_name='` + t_name + `'
      order by
        ordinal_position;
      `

		rows, err := DB.Query(query)
		checkError(err)

		var model_str string
    var needTimePackage bool
		for rows.Next() {
			var (
				column_name    string
				data_type      string
				column_default string
				is_nullable    string
			)

			err = rows.Scan(&column_name, &data_type, &column_default, &is_nullable)
			checkError(err)

			json := genj(column_name, column_default, primary_key)

      if data_type == "timestamp with time zone" {
        needTimePackage = true
      }

			// If have to use pointer
			if data_type == "timestamp with time zone" && is_nullable == "YES" {
				if hasNullRecoreds(t_name, column_name) == true {
					data_type = "*time.Time"
				}
			}

			m := gormColName(column_name) + " " + gormDataType(data_type) + " `" + json + "`\n"
			model_str += m

			isInfered, inf_column_name := inferORM(column_name)

      // Add belongs_to relation
			if isInfered == true {
				json := genj(strings.ToLower(inf_column_name), "", nil)
				comment := "// This line is infered from column name \"" + column_name + "\"."
				inf_column_name = gormColName(inf_column_name)

				m := inf_column_name + " *" + inf_column_name + " `" + json + "` " + comment + "\n"
				model_str += m
			}
		}

    var importPackage string
    if needTimePackage {
      importPackage = "import \"time\"\n\n"
    } else {
      importPackage = ""
    }

		model_str = "package models\n\n" + importPackage + "type " + gormTableName(t_name) + " struct {\n" + model_str + "}\n"

		// fmt.Println(model_str) // Print output

		file, err := os.Create(OutDir + inflector.Singularize(t_name) + `.go`)
		checkError(err)
		defer file.Close()
		file.Write(([]byte)(model_str))
	}

	err := exec.Command("gofmt", "-w", "models").Run()
	checkError(err)
}

// Infer belongs_to Relation from column's name
func inferORM(s string) (bool, string) {
	s = strings.ToLower(s)
	ss := strings.Split(s, "_")

	const (
		id = "id"
	)

	var new_ss []string
	var containsID bool = false
	for _, word := range ss {
		if word == id {
			containsID = true
			continue
		}

		new_ss = append(new_ss, word)
	}

	if containsID == false || len(new_ss) == 0 {
		return false, ""
	}

	inf_column_name := strings.Join(new_ss, "_")
	return true, inf_column_name
}

// Generate json
func genj(column_name, column_default string, primary_key map[string]bool) (json string) {
	json = "json:\"" + column_name + "\""

	if primary_key[column_name] == true {
		p := "gorm:\"primary_key;AUTO_INCREMENT\" "
		json = p + json
	}

	if column_default != "" && !strings.Contains(column_default, "nextval") {
		d := " sql:\"DEFAULT:" + column_default + "\""
		json += d
	}

	return
}

func hasNullRecoreds(table_name string, column_name string) bool {
	query := `SELECT COUNT(*) FROM ` + table_name + ` WHERE ` + column_name + ` IS NULL;`

	var count string

	err := DB.QueryRow(query).Scan(&count)
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
        OutDir = c.Args()[1] + "/"
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

    		db, err := sql.Open("postgres", paramFirst)
    		DB = db
    		checkError(err)
    		defer DB.Close()
    		tables := getTableName()

    		fmt.Println("Generating gorm from tables below...")
        for _, table_name := range tables {
          fmt.Printf("Table name: %s\n", table_name)
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
