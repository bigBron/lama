package lama

import (
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var Conf Cfg

type Cfg = *viper.Viper

type Config struct {
	Conf Cfg
}

func GetWorkerDir() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func (s *Config) Init() error {
	s.Conf = NewConf()
	return nil
}

func NewConf() Cfg {
	if Conf == nil {
		Conf = viper.New()
		Conf.SetConfigName("cfg")
		Conf.SetConfigType("toml")
		Conf.AddConfigPath(GetWorkerDir())
		err := Conf.ReadInConfig()
		if err != nil {
			log.Fatal(err)
		}
	}
	return Conf
}

func (s *Config) Provide() Cfg {
	return s.Conf
}

func (s *Config) Provides() []any {
	return []interface{}{
		s.Provide,
	}
}
