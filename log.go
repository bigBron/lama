package lama

import (
	"github.com/kataras/golog"
)

var Print Log

type Log = *golog.Logger

type Logger struct {
	log Log
}

func (s *Logger) Init() error {
	s.log = NewLog()
	return nil
}

func NewLog() Log {
	if Print == nil {
		Print = golog.Default
		level := "debug"
		if Conf != nil {
			level = Conf.String("app.logLevel")
		}
		Print.SetLevel(level)
	}
	return Print
}

func (s *Logger) ProvideLog() Log {
	return s.log
}

func (s *Logger) Provides() []any {
	return []any{
		s.ProvideLog,
	}
}
