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

type Web struct {
	services []any
}

// NewWeb 实例化web服务
func NewWeb() *Web {
	return &Web{
		services: []any{
			&provide{},
			&Act{},
			&Http{},
		},
	}
}

// Register 注册服务
func (s *Web) Register(services ...any) *Web {
	for _, srv := range s.services {
		services = append(services, srv)
	}
	s.services = services
	return s
}

// Run 运行web服务
func (s *Web) Run() {
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
		case e := <-errCh:
			if e != nil {
				Print.Debug(e.Error)
				er := app.Stop() // 出现错误，停止服务
				if er != nil {
					Print.Fatal(er)
				}
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
