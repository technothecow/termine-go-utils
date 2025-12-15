package httpmw

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// Logging returns an Echo middleware that logs incoming HTTP requests using
// the provided zap.Logger.
//
// Fields:
//   - method
//   - path
//   - status
//   - latency
//   - remote_ip
//   - user_agent
//   - request_id
//   - error (only when non-nil)
func Logging(logger *zap.Logger) echo.MiddlewareFunc {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()

			start := time.Now()
			err := next(c)
			stop := time.Now()

			latency := stop.Sub(start)

			fields := []zap.Field{
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.Int("status", res.Status),
				zap.Duration("latency", latency),
				zap.String("remote_ip", c.RealIP()),
				zap.String("user_agent", req.UserAgent()),
				zap.String("request_id", req.Header.Get(echo.HeaderXRequestID)),
			}

			if err != nil {
				// Ensure Echo's HTTP error handling still runs.
				c.Error(err)
				fields = append(fields, zap.Error(err))
				logger.Error("http request failed", fields...)
				return err
			}

			logger.Info("http request", fields...)
			return nil
		}
	}
}

// Recovery wraps Echo's Recover middleware and directs panic information
// (including stack traces) into the provided zap.Logger.
func Recovery(logger *zap.Logger) echo.MiddlewareFunc {
	if logger == nil {
		logger = zap.NewNop()
	}

	return middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize:         1 << 10, // 1KB
		DisableStackAll:   false,
		DisablePrintStack: true,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			logger.Error("panic recovered in http handler",
				zap.Error(err),
				zap.ByteString("stack", stack),
			)
			return err
		},
	})
}
