package main

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/gedex/inflector"
	"github.com/serenize/snaker"
)

func GenModel(tableName string, pkeys map[string]bool, fields []*Field, outPath string) error {
	var gormStr string
	var needTimePackage bool

	for _, field := range fields {
		json := genJSON(field.Name, field.Default, pkeys)
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

		m := gormColName(field.Name) + " " + fieldType + " `" + json + "`\n"
		gormStr += m

		isInfered, infColName := inferORM(field.Name)

		// Add belongs_to relation
		if isInfered {
			json := genJSON(strings.ToLower(infColName), "", nil)
			comment := "// This line is infered from column name \"" + field.Name + "\"."
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

// Infer belongs_to Relation from column's name
func inferORM(s string) (bool, string) {
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
func gormColName(s string) string {
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
