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
			r.Init(App)
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
}

func (s *Action) Provide() (IRISApp, NewMvcApp, Version, Deprecated, NewParty, NewMvc) {
	app := NewIRISApp()
	newMvc := func(path string) MVCApp {
		return mvc.New(app.APIBuilder.Party(path))
	}
	return app,
		mvc.New,
		mvc.Version,
		mvc.Deprecated,
		app.APIBuilder.Party,
		newMvc
}
