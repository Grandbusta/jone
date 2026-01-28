package config

import (
	"testing"
	"time"
)

func TestPool_ZeroValuePreservesDefaults(t *testing.T) {
	var p Pool

	if p.MaxOpenConns != 0 {
		t.Errorf("MaxOpenConns = %d, want 0", p.MaxOpenConns)
	}
	if p.MaxIdleConns != 0 {
		t.Errorf("MaxIdleConns = %d, want 0", p.MaxIdleConns)
	}
	if p.ConnMaxLifetime != 0 {
		t.Errorf("ConnMaxLifetime = %v, want 0", p.ConnMaxLifetime)
	}
	if p.ConnMaxIdleTime != 0 {
		t.Errorf("ConnMaxIdleTime = %v, want 0", p.ConnMaxIdleTime)
	}
}

func TestPool_CustomValues(t *testing.T) {
	p := Pool{
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	if p.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns = %d, want 25", p.MaxOpenConns)
	}
	if p.MaxIdleConns != 10 {
		t.Errorf("MaxIdleConns = %d, want 10", p.MaxIdleConns)
	}
	if p.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want %v", p.ConnMaxLifetime, 30*time.Minute)
	}
	if p.ConnMaxIdleTime != 5*time.Minute {
		t.Errorf("ConnMaxIdleTime = %v, want %v", p.ConnMaxIdleTime, 5*time.Minute)
	}
}

func TestConfig_PoolFieldExists(t *testing.T) {
	cfg := Config{
		Client: "postgresql",
		Connection: Connection{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "password",
			Database: "testdb",
		},
		Pool: Pool{
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
		},
		Migrations: Migrations{
			TableName: "jone_migrations",
		},
	}

	if cfg.Pool.MaxOpenConns != 10 {
		t.Errorf("Config.Pool.MaxOpenConns = %d, want 10", cfg.Pool.MaxOpenConns)
	}
	if cfg.Pool.MaxIdleConns != 5 {
		t.Errorf("Config.Pool.MaxIdleConns = %d, want 5", cfg.Pool.MaxIdleConns)
	}
	if cfg.Pool.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("Config.Pool.ConnMaxLifetime = %v, want %v", cfg.Pool.ConnMaxLifetime, 30*time.Minute)
	}
	if cfg.Pool.ConnMaxIdleTime != 5*time.Minute {
		t.Errorf("Config.Pool.ConnMaxIdleTime = %v, want %v", cfg.Pool.ConnMaxIdleTime, 5*time.Minute)
	}
}
