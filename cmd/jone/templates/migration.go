package templates

import "text/template"

// MigrationStubData holds data for the migration stub template.
type MigrationStubData struct {
	RuntimePackage string
}

const migrationTemplateContent = `package migration

import (
	"{{ .RuntimePackage }}"
)

func Up(s jone.Schema) {

}

func Down(s jone.Schema) {

}
`

// Migration is the parsed template for generating migration stub files.
var Migration = template.Must(template.New("migration").Parse(migrationTemplateContent))

// RenderMigration generates the migration.go stub content.
func RenderMigration(data MigrationStubData) ([]byte, error) {
	return Render(Migration, data)
}
