package lama

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/middleware/accesslog"
	"github.com/kataras/iris/v12/middleware/cors"
	"github.com/kataras/iris/v12/middleware/requestid"
	"github.com/kataras/iris/v12/mvc"
)

type NewMvc func(string) MVCApp
type NewMvcApp func(router.Party) MVCApp
type NewParty func(string, ...iris.Handler) router.Party

type Version func(string) mvc.OptionFunc
type Deprecated func(mvc.DeprecationOptions) mvc.OptionFunc

type UseGlobalMiddleware interface {
	UseGlobal() iris.Handler
}

type Act *Action
type IRISApp = *iris.Application
type MVCApp = *mvc.Application

var App IRISApp

func NewIRISApp() IRISApp {
	if App == nil {
		App = iris.New()

		if Conf.Bool("app.accesslog") {
			App.UseRouter(accesslog.New(Print.Printer).Handler)
		}

		if Conf.Bool("app.recover") {
			r := &Recover{debug: Conf.Bool("app.debug")}
			App.UseRouter(r.UseGlobal())
		}

		App.UseRouter(requestid.New())
		App.UseRouter(cors.New().
			ExtractOriginFunc(cors.DefaultOriginExtractor).
			ReferrerPolicy(cors.NoReferrerWhenDowngrade).
			AllowOriginFunc(cors.AllowAnyOrigin).
			Handler())

		var disableStartupLog bool
		debug := Conf.Bool("app.debug")
		if !debug {
			disableStartupLog = true
		}

		conf := []iris.Configurator{
			iris.WithoutInterruptHandler,
			iris.WithoutServerError(iris.ErrServerClosed),
			iris.WithConfiguration(iris.Configuration{
				PostMaxMemory:     100 << 20,
				DisableStartupLog: disableStartupLog,
			}),
		}

		App.Configure(conf...)
		App.AllowMethods(iris.MethodOptions)
	}
	return App
}

type Action struct {
	IRISApp IRISApp
}

func (s *Action) Init() error {
	s.IRISApp = NewIRISApp()
	return nil
}

func (s *Action) UseGlobalMiddleware(middle UseGlobalMiddleware) {
	s.IRISApp.UseGlobal(middle.UseGlobal())
}

func (s *Action) Collects() []any {
	return []any{
		s.UseGlobalMiddleware,
	}
}

func (s *Action) ProvideApp() IRISApp {
	return s.IRISApp
}

func (s *Action) ProvideVersion() Version {
	return mvc.Version
}

func (s *Action) ProvideDeprecated() Deprecated {
	return mvc.Deprecated
}

func (s *Action) ProvideNewParty() NewParty {
	return s.IRISApp.APIBuilder.Party
}

func (s *Action) ProvideNewMvc() NewMvc {
	return func(path string) MVCApp {
		return mvc.New(s.IRISApp.APIBuilder.Party(path))
	}
}

func (s *Action) ProvideNewMvcApp() NewMvcApp {
	return mvc.New
}

func (s *Action) ProvideAct() Act {
	return s
}

func (s *Action) Provides() []any {
	return []any{
		s.ProvideApp,
		s.ProvideVersion,
		s.ProvideDeprecated,
		s.ProvideNewParty,
		s.ProvideNewMvc,
		s.ProvideNewMvcApp,
		s.ProvideAct,
	}
}
