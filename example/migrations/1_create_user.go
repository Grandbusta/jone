package migrations

import "github.com/Grandbusta/jone"

func Up(j *jone.Jone) {
	j.Schema.CreateTable("users", func(table *jone.Table) {
		table.Int("id")
		table.String("name")
		table.Bool("is_admin")
		table.DropColumn("food")
	})
}

func Down(j *jone.Jone) {
	j.Schema.DropTable("users")
	j.Schema.Table("users", func(table *jone.Table) {
		table.RenameColumn("id", "user_id")
		table.RenameColumn("name", "username")
		table.RenameColumn("age", "user_age")
	})
}
