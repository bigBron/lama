package lama

import (
	"context"
	"fmt"
)

func init() {
	NewLog()
}

type Http struct {
}

// Serve 启动核心
func (s *Http) Serve() chan error {
	errCh := make(chan error, 1)
	Print.Info(fmt.Sprintf("App Version %s", Conf.String("app.version")))

	go func() {
		addr := Conf.String("app.addr")
		Print.Infof("HTTP Server Listening On http://localhost%s", addr)
		err := NewIRISApp().Listen(addr)
		if err != nil {
			errCh <- err
		}
	}()

	return errCh
}

func (s *Http) Stop() error {
	Print.Info("HTTP Server Shutdown Gracefully")
	return NewIRISApp().Shutdown(context.Background())
}
