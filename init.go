package lama

import (
	gookit "github.com/gookit/config/v2"
	"github.com/gookit/config/v2/toml"
	"github.com/kataras/golog"
	"log"
	"os"
	"path/filepath"
)

func init() {
	newCfg()
	newLog()
}

type Log = *golog.Logger
type Cfg = *gookit.Config

var Conf Cfg
var Print Log

type provide struct {
}

func (s *provide) Provide() (Cfg, Log) {
	return Conf, Print
}

func newLog() Log {
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

func newCfg() Cfg {
	if Conf == nil {
		gookit.AddDriver(toml.Driver)
		Conf = gookit.Default()
		Conf.WithOptions(func(opt *gookit.Options) {
			opt.DecoderConfig.TagName = "json"
		})
		Conf.LoadFiles(GetWorkerDir() + "/cfg.json")
	}
	return Conf
}

func GetWorkerDir() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}
