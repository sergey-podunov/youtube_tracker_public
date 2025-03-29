//go:build atlas_schema

package main

import (
	"log"
	"os"
	"strings"
	"text/template"
)

type TemplateData struct {
	SQL string
}

func main() {
	templateFile := "database/atlas-schema.yaml.tpl"
	sqlFile := "database/schema.sql"
	destinationFile := "database/atlas-schema.yaml"

	sqlContent, err := os.ReadFile(sqlFile)
	if err != nil {
		log.Fatalf("Error reading SQL file: %v", err)
	}

	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		log.Fatalf("Error parsing template file: %v", err)
	}

	data := TemplateData{
		SQL: indentSQL(string(sqlContent), 6),
	}

	outputFile, err := os.Create(destinationFile)
	if err != nil {
		log.Fatalf("Error creating output file: %v", err)
	}
	defer outputFile.Close()

	err = tmpl.Execute(outputFile, data)
	if err != nil {
		log.Fatalf("Error executing the template: %v", err)
	}

	log.Printf("%s has been successfully generated.\n", destinationFile)
}

func indentSQL(sql string, spaces int) string {
	indent := strings.Repeat(" ", spaces)

	sqlLines := strings.Split(sql, "\n")
	indentedSQL := make([]string, len(sqlLines))
	for num, line := range sqlLines {
		indentedSQL[num] += indent + line
	}

	return strings.Join(indentedSQL, "\n")
}
