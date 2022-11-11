package lama

import (
	gookit "github.com/gookit/config/v2"
)

var Conf Cfg

type Cfg = *gookit.Config

type Config struct {
	Conf Cfg
}

func (s *Config) Init() error {
	s.Conf = Conf
	return nil
}

func (s *Config) Provide() Cfg {
	return s.Conf
}

func (s *Config) Provides() []any {
	return []interface{}{
		s.Provide,
	}
}
