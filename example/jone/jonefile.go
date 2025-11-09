package jone

import "github.com/Grandbusta/jone"

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
	