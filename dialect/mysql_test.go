package dialect

import (
	"testing"

	"github.com/Grandbusta/jone/config"
)

func TestMySQLDialect_Name(t *testing.T) {
	d := &MySQLDialect{}
	if got := d.Name(); got != "mysql" {
		t.Errorf("Name() = %q, want %q", got, "mysql")
	}
}

func TestMySQLDialect_DriverName(t *testing.T) {
	d := &MySQLDialect{}
	if got := d.DriverName(); got != "mysql" {
		t.Errorf("DriverName() = %q, want %q", got, "mysql")
	}
}

func TestMySQLDialect_FormatDSN(t *testing.T) {
	d := &MySQLDialect{}

	tests := []struct {
		name string
		conn config.Connection
		want string
	}{
		{
			name: "basic connection",
			conn: config.Connection{
				Host:     "localhost",
				Port:     "3306",
				User:     "root",
				Password: "secret",
				Database: "testdb",
			},
			want: "root:secret@tcp(localhost:3306)/testdb",
		},
		{
			name: "remote host",
			conn: config.Connection{
				Host:     "db.example.com",
				Port:     "3307",
				User:     "admin",
				Password: "pass",
				Database: "prod",
			},
			want: "admin:pass@tcp(db.example.com:3307)/prod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.FormatDSN(tt.conn)
			if got != tt.want {
				t.Errorf("FormatDSN() = %q, want %q", got, tt.want)
			}
		})
	}
}
