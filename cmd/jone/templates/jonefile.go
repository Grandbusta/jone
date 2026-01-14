package templates

import "text/template"

// JoneFileData holds data for the jonefile template.
type JoneFileData struct {
	RuntimePackage string
}

const jonefileTemplateContent = `package jone

import "{{ .RuntimePackage }}"

var Config = jone.Config{
	Client:     "postgresql",
	Connection: jone.Connection{
		User:     "username",
		Password: "password",
		Database: "my_db",
	},
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
