package main

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gedex/inflector"
	"github.com/serenize/snaker"
)

type TemplateField struct {
	Name    string
	Type    string
	Tag     string
	Comment string
}

type TemplateParams struct {
	Name            string
	Fields          []*TemplateField
	NeedTimePackage bool
}

var hasMany = make(map[string][]string)

func GenerateModel(table string, pkeys map[string]bool, fields []*Field, tables []string) *TemplateParams {
	var needTimePackage bool

	templateFields := []*TemplateField{}

	for _, field := range fields {
		fieldType := gormDataType(field.Type)

		if fieldType == "time.Time" || fieldType == "*time.Time" {
			needTimePackage = true

			if field.Nullable {
				fieldType = "*time.Time"
			} else {
				fieldType = "time.Time"
			}
		}

		if fieldType == "double precision" {
			fieldType = "float32"
		}

		templateFields = append(templateFields, &TemplateField{
			Name: gormColumnName(field.Name),
			Type: fieldType,
			Tag:  genJSON(field.Name, field.Default, pkeys),
		})

		isInfered, infColName := inferORM(field.Name, tables)

		colName := gormColumnName(infColName)

		// Add belongs_to relation
		if isInfered {
			templateFields = append(templateFields, &TemplateField{
				Name:    colName,
				Type:    "*" + colName,
				Tag:     genJSON(strings.ToLower(infColName), "", nil),
				Comment: "This line is infered from column name \"" + field.Name + "\".",
			})

			// Add has_many relation
			hasMany[colName] = append(hasMany[colName], table)
		}
	}

	params := &TemplateParams{
		Name:            gormTableName(table),
		Fields:          templateFields,
		NeedTimePackage: needTimePackage,
	}

	return params
}

func AddHasMany(params *TemplateParams) {
	if _, ok := hasMany[params.Name]; ok {
		for _, infColName := range hasMany[params.Name] {
			params.Fields = append(params.Fields, &TemplateField{
				Name:    gormColumnName(infColName),
				Type:    "[]*" + gormTableName(infColName),
				Tag:     genJSON(strings.ToLower(infColName), "", nil),
				Comment: "This line is infered from other tables.",
			})
		}
	}
}

func SaveModel(table string, params *TemplateParams, outPath string) error {
	body, err := Asset("_templates/model.go.tmpl")
	if err != nil {
		return err
	}

	tmpl, err := template.New("").Parse(string(body))
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, params); err != nil {
		return err
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	modelFile := filepath.Join(outPath, inflector.Singularize(table)+".go")

	if err := ioutil.WriteFile(modelFile, src, 0644); err != nil {
		return err
	}

	return nil
}

// Infer belongs_to Relation from column's name
func inferORM(s string, tables []string) (bool, string) {
	s = strings.ToLower(s)
	ss := strings.Split(s, "_")

	newSS := []string{}
	var containsID bool = false
	for _, word := range ss {
		if word == "id" {
			containsID = true
			continue
		}

		newSS = append(newSS, word)
	}

	if containsID == false || len(newSS) == 0 {
		return false, ""
	}

	infColName := strings.Join(newSS, "_")

	// Check the table is existed or not
	tableName := snaker.CamelToSnake(infColName)
	tableName = inflector.Pluralize(tableName)

	exist := false
	for _, table := range tables {
		if table == tableName {
			exist = true
		}
	}

	if !exist {
		return false, ""
	}

	return true, infColName
}

// Generate json
func genJSON(columnName, columnDefault string, primaryKeys map[string]bool) (json string) {
	json = "json:\"" + columnName + "\""

	if primaryKeys[columnName] {
		p := "gorm:\"primary_key;AUTO_INCREMENT\" "
		json = p + json
	}

	if columnDefault != "" && !strings.Contains(columnDefault, "nextval") {
		d := " sql:\"DEFAULT:" + columnDefault + "\""
		json += d
	}

	return
}

// Singlarlize table name and upper initial character
func gormTableName(s string) string {
	var tableName string

	tableName = strings.ToLower(s)
	tableName = inflector.Singularize(tableName)
	tableName = snaker.SnakeToCamel(tableName)

	return strings.Title(tableName)
}

// Ex: facebook_uid → FacebookUID
func gormColumnName(s string) string {
	s = strings.ToLower(s)
	ss := strings.Split(s, "_")

	for i, word := range ss {
		if word == "id" || word == "uid" || word == "url" {
			word = strings.ToUpper(word)
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
	case "timestamp with time zone", "timestamp without time zone":
		return "time.Time"
	case "date":
		return "*time.Time"
	default:
		return s
	}
}
