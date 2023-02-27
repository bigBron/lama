package lama

import (
	"github.com/gookit/validate"
	"github.com/kataras/iris/v12/context"
	"time"
)

type Recover struct {
	debug bool
}

func (s *Recover) Init(app IRISApp) error {
	app.UseGlobal(func(ctx *context.Context) {
		defer func() {
			if err := recover(); err != nil {
				if ctx.IsStopped() { // handled by other middleware.
					return
				}

				var code int
				var msg string
				ret := map[string]any{
					"state": false,
					"time":  time.Now().Unix(),
				}

				switch e := err.(type) {
				case string:
					code = 400
					msg = e

				case validate.Errors:
					code = 400
					msg = e.One()

				default:
					code = 500
					msg = "Server internal error"
					if s.debug {
						// todo 优化
						msg = err.(error).Error()
					}
				}

				ret["msg"] = msg
				ctx.StopWithJSON(code, ret)
			}
		}()
		ctx.Next()
	})
	return nil
}
