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
	Host     string `json:"host" validate:"required"`
	Port     int    `json:"port" validate:"required"`
	User     string `json:"user" validate:"required"`
	Passwd   string `json:"passwd" validate:"required"`
	DBName   string `json:"dbname" validate:"required"`
	MinConn  int    `json:"minConn" validate:"required"`
	PoolSize int    `json:"poolSize" validate:"required"`
	Timeout  int    `json:"timeout" validate:"required"`
	Timezone string `json:"timezone" validate:"required"`
}

func (s *PG) Provide() SqlxDB {
	err := Conf.Structure("pg", &s.cfg)
	if err != nil {
		panic(err)
	}

	v := validate.New(&s.cfg)
	if !v.Validate() {
		panic(fmt.Errorf("postgresql config[pg]: %s", v.Errors.One()))
	}

	s.dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s TimeZone=%s sslmode=disable", s.cfg.Host, s.cfg.Port, s.cfg.User, s.cfg.Passwd, s.cfg.DBName, s.cfg.Timezone)

	db, err := sqlx.Open("postgres", s.dsn)
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	db.SetMaxIdleConns(s.cfg.MinConn)
	db.SetMaxOpenConns(s.cfg.PoolSize)
	db.SetConnMaxLifetime(time.Second * time.Duration(s.cfg.Timeout))

	s.db = db
	Print.Infof("Connected Postgresql %s", s.cfg.Host)

	DefaultDB = PGSQL
	return db
}

func (s *PG) Stop() error {
	Print.Infof("Disconnect Postgresql %s", s.cfg.Host)
	return s.db.Close()
}
