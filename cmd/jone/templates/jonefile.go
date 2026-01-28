package templates

import "text/template"

// JoneFileData holds data for the jonefile template.
type JoneFileData struct {
	RuntimePackage string
	Database       string
}

const jonefileTemplateContent = `package jone

import (
	"{{ .RuntimePackage }}"

{{- if eq .Database "mysql" }}
	// Database driver
	_ "github.com/go-sql-driver/mysql"
{{- else if eq .Database "sqlite" }}
	// Database driver
	_ "github.com/mattn/go-sqlite3"
{{- else }}
	// Database driver
	_ "github.com/jackc/pgx/v5/stdlib"
{{- end }}
)

var Config = jone.Config{
{{- if eq .Database "mysql" }}
	Client:     "mysql",
	Connection: jone.Connection{
		Host:     "localhost",
		Port:     "3306",
		User:     "root",
		Password: "password",
		Database: "my_db",
	},
{{- else if eq .Database "sqlite" }}
	Client:     "sqlite3",
	Connection: jone.Connection{
		Database: "./jone.db",
	},
{{- else }}
	Client:     "postgresql",
	Connection: jone.Connection{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "password",
		Database: "my_db",
	},
{{- end }}
	Migrations: jone.Migrations{
		TableName: "jone_migrations",
	},
}
`

// JoneFile is the parsed template for generating jonefile.go.
var JoneFile = template.Must(template.New("jonefile").Parse(jonefileTemplateContent))

// RenderJoneFile generates the jonefile.go content.
func RenderJoneFile(data JoneFileData) ([]byte, error) {
	return Render(JoneFile, data)
}
