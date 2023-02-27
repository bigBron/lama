package lama

import (
	"os"
	"os/signal"
	"syscall"
)

/**
// 服务接口
type Services interface (
	// 提供依赖项，可以返回一个或者多个依赖项，可选
	Provide() provide,...

	// 初始化服务，可选
	Init() error

	// 启动服务时候调用，可选
	Serve() error

	// 关闭服务时候调用，可选
	Stop() error
)
*/

type Srv struct {
	services []any
}

// NewSrv 实例化server服务
func NewSrv() *Srv {
	return &Srv{
		services: []any{
			&provide{},
		},
	}
}

// Register 注册服务
func (s *Srv) Register(services ...any) *Srv {
	for _, srv := range services {
		s.services = append(s.services, srv)
	}
	return s
}

// Run 运行服务
func (s *Srv) Run() {
	app := NewAda()
	err := app.Register(s.services...)
	if err != nil {
		Print.Fatal(err)
	}

	// 初始化服务
	err = app.Init()
	if err != nil {
		Print.Fatal(err)
	}

	// 启动服务
	errCh := app.Serve()

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
			er := app.Stop()
			if er != nil {
				Print.Fatal(er)
			}
			os.Exit(0)
			return
		}
	}
}
