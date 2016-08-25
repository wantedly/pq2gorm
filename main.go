package main

import (
  "fmt"
  "strings"
  "os"
  "net"
  nurl "net/url"
  _ "github.com/lib/pq"
  "database/sql"
)

var DB, ERR = sql.Open("postgres", "postgres://admin:@localhost:5432/visit?sslmode=disable")

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

func writeModel(t_names []string){
  for _, t_name := range t_names {

    // query := `
    //   select column_name, data_type, COALESCE(column_default, '') as column_default
    //   from information_schema.columns
    //   where
    //   table_catalog='` + `visit` + `'
    //   and
    //   table_name='` + t_name + `'
    //   order by
    //   ordinal_position;
    //   `
    query := `
      select column_name, data_type, COALESCE(column_default, '') as column_default
      from information_schema.columns
      where
      table_name='` + t_name + `'
      order by
      ordinal_position;
      `

    fmt.Println(query)
    rows, err := DB.Query(query)
    checkError(err)

    model_str := "type " + gormTableName(t_name) + " struct {\n"
    for rows.Next() {
      var (
        column_name string
        data_type string
        column_default string
      )
      err = rows.Scan(&column_name, &data_type, &column_default)
      checkError(err)
      m := gormColName(column_name) + " " + gormDataType(data_type) + "\n"
      model_str += m
    }

    model_str = "package models\n\nimport \"time\"\n\n" + model_str + "}"

    fmt.Println(model_str)
    file, err := os.Create(`models/` + t_name + `.go`)
    checkError(err)
    defer file.Close()
    file.Write(([]byte)(model_str))
  }
}

// Singlarlize table name and upper initial character
func gormTableName(s string) string {
  s = strings.ToLower(s)
  if strings.HasSuffix(s, "s") {
    s = string([]rune(s)[:len(s)-1])
  }
  return strings.Title(s)
}

// Ex: facebook_uid → FacebookUID
func gormColName(s string) string {
  s = strings.ToLower(s)
  ss := strings.Split(s, "_")
  const (
    id = "id"
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
  case "character", "text":
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
func ParseURL(url string) (map[string]string, error) {
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
  checkError(ERR)
  defer DB.Close()
  test := getTableName()
  //a, _ := ParseURL("postgres://admin:@localhost:5432/visit?sslmode=disable")
  //fmt.Println(a)
  fmt.Println(test)
  writeModel(test)
}
