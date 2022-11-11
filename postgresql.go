package lama

import (
	"fmt"
	"github.com/gookit/validate"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"time"
)

type PG struct {
	dsn string
	cfg PGConf
	db  SqlxDB
}

type PGConf struct {
	Host     string `validate:"required"`
	Port     int    `validate:"required"`
	User     string `validate:"required"`
	Passwd   string `validate:"required"`
	DBName   string `validate:"required"`
	MinConn  int    `validate:"required"`
	PoolSize int    `validate:"required"`
	Timeout  int    `validate:"required"`
	Timezone string `validate:"required"`
}

func (s *PG) Init() error {
	err := Conf.Structure("pg", &s.cfg)
	if err != nil {
		return err
	}

	v := validate.New(&s.cfg)
	if !v.Validate() {
		return fmt.Errorf("postgresql config[pg]: %s", v.Errors.One())
	}

	s.dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s TimeZone=%s sslmode=disable", s.cfg.Host, s.cfg.Port, s.cfg.User, s.cfg.Passwd, s.cfg.DBName, s.cfg.Timezone)

	db, err := sqlx.Open("postgres", s.dsn)
	if err != nil {
		return err
	}

	err = db.Ping()
	if err != nil {
		return err
	}

	db.SetMaxIdleConns(s.cfg.MinConn)
	db.SetMaxOpenConns(s.cfg.PoolSize)
	db.SetConnMaxLifetime(time.Second * time.Duration(s.cfg.Timeout))

	s.db = db
	Print.Infof("Connected Postgresql %s", s.cfg.Host)

	DefaultDB = PGSQL
	return nil
}

func (s *PG) Serve() chan error {
	return make(chan error)
}

func (s *PG) Stop() error {
	Print.Infof("Disconnect Postgresql %s", s.cfg.Host)
	return s.db.Close()
}

func (s *PG) Provide() SqlxDB {
	return s.db
}

func (s *PG) Provides() []any {
	return []interface{}{
		s.Provide,
	}
}
