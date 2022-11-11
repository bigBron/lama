package lama

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	endure "github.com/roadrunner-server/endure/pkg/container"
)

/**
容器接口说明：
type (
	// 服务生命周期接口
	Service interface {
		// 启动服务时候调用
		Serve() chan error
		// 关闭服务时候调用
		Stop() error
	}

	// 服务名称
	Named interface {
		Name() string
	}

	// 提供额外依赖，返回一个函数列表
	// 函数接受调用者名称参数，并返回依赖跟error，error可以省略
	Provider interface {
		Provides() []interface{ fn(name endure.Named) (指针依赖, error) }
	}

	// 获取容器中匹配接口实例，返回一个函数列表
	// 函数接受匹配的依赖，并返回error
	Collector interface {
		Collects() []interface{ fn(依赖) error }
	}
)

// 插件接口
type Plugin struct{}

// 初始化方法，必须实现
func (p *Plugin) Init( ) error {
	return nil
}
*/

type Web struct {
	list []any
}

// NewWeb 实例化web服务
func NewWeb() *Web {
	return &Web{
		list: []any{
			&Http{},
			&Action{},
			&Config{},
			&Logger{},
		},
	}
}

// Register 注册服务
func (s *Web) Register(serv any) *Web {
	s.list = append(s.list, serv)
	return s
}

// RegisterAll 批量注册服务
func (s *Web) RegisterAll(servList []any) *Web {
	s.list = append(
		s.list,
		servList...,
	)
	return s
}

// Run 运行web服务
func (s *Web) Run() {
	logLevel := endure.ErrorLevel
	if Conf.Bool("app.debugContainer") {
		logLevel = endure.DebugLevel
	}

	// 创建容器
	timeout := time.Duration(Conf.Int64("app.shutdownTimeout"))
	app, err := endure.NewContainer(nil, endure.SetLogLevel(logLevel), endure.GracefulShutdownTimeout(time.Second*timeout))
	if err != nil {
		Print.Fatal(err)
	}

	// 注册服务
	err = app.RegisterAll(s.list...)
	if err != nil {
		Print.Fatal(err)
	}

	// 初始化服务
	err = app.Init()
	if err != nil {
		Print.Fatal(err)
	}

	// 启动服务
	errCh, err := app.Serve()
	if err != nil {
		Print.Fatal(err)
	}

	// 监听中断信号
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errCh:
			Print.Debug(err.Error)
			er := app.Stop() // 出现错误，停止服务
			if er != nil {
				Print.Fatal(er)
			}
		case <-stop:
			er := app.Stop() // 停止服务
			if er != nil {
				Print.Fatal(er)
			}
			os.Exit(0)
			return
		}
	}
}
