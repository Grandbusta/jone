package jone

import "github.com/Grandbusta/jone/schema"

type Migration interface {
	Up(j *Jone)
	Down(j *Jone)
}

type Config struct {
	User string
	Pass string
	Host string
	Port int
	DB   string
}

type Jone struct {
	*schema.Schema
	*schema.Table
}
