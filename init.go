package lama

import (
	gookit "github.com/gookit/config/v2"
	"github.com/gookit/config/v2/toml"
	"log"
	"os"
	"path/filepath"
)

func init() {
	initCfg()
}

func GetWorkerDir() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func initCfg() {
	gookit.AddDriver(toml.Driver)
	if Conf == nil {
		Conf = gookit.Default()

		cfgF := GetWorkerDir() + "/cfg.toml"
		err := Conf.LoadFiles(cfgF)

		if err != nil {
			log.Fatal(err)
		}
	}
}
