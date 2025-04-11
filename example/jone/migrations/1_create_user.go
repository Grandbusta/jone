package migrations

import "github.com/Grandbusta/jone"

func Up(j *jone.Jone) {
	j.Schema.CreateTable("users", func() {
		j.Table.Int("id")
		j.Table.String("name")
		j.Table.Bool("is_admin")
		j.Table.DropColumn("food")
	})
}

func Down(j *jone.Jone) {
	j.Schema.DropTable("users")
	j.Schema.Table("users", func() {
		j.Table.RenameColumn("id", "user_id")
		j.Table.RenameColumn("name", "username")
		j.Table.RenameColumn("age", "user_age")
	})
}
