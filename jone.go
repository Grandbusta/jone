package jone

type Config struct {
	Client     string
	Connection Connection
	Migrations Migrations
}

type Connection struct {
	User     string
	Password string
	Database string
	Port     string
	Host     string
}

type Migrations struct {
	TableName string
}
