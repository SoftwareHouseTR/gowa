package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func Recovery() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var nextErr error
		panicVal := catchPanic(func() {
			nextErr = ctx.Next()
		})

		if panicVal == nil {
			return nextErr
		}

		// Error 479: WhatsApp rate limit — retry once after 5 seconds
		errMsg := fmt.Sprintf("%v", panicVal)
		if strings.Contains(errMsg, "server returned error 479") {
			retryCount, _ := ctx.Locals("__retry_479").(int)
			if retryCount < 1 {
				ctx.Locals("__retry_479", retryCount+1)
				logrus.Warnf("[RATE_LIMIT] Error 479 detected, retrying in 5s...")
				time.Sleep(5 * time.Second)
				return ctx.RestartRouting()
			}
		}

		var res utils.ResponseData
		res.Status = 500
		res.Code = "INTERNAL_SERVER_ERROR"
		res.Message = errMsg

		// Check for typed errors first
		if ctxErr, ok := panicVal.(error); ok && ctxErr == context.DeadlineExceeded {
			res.Status = 504
			res.Code = "GATEWAY_TIMEOUT"
			res.Message = "Request timed out waiting for WhatsApp server response"
		}

		if errGeneric, ok := panicVal.(pkgError.GenericError); ok {
			res.Status = errGeneric.StatusCode()
			res.Code = errGeneric.ErrCode()
			res.Message = errGeneric.Error()
		} else if strings.Contains(strings.ToLower(res.Message), "not found") {
			res.Status = 404
			res.Code = "NOT_FOUND"
		}

		if res.Status >= 500 {
			logrus.Errorf("Panic recovered in middleware: %v", panicVal)
		} else {
			logrus.Warnf("Recovered in middleware: %v", panicVal)
		}

		return ctx.Status(res.Status).JSON(res)
	}
}

// catchPanic executes fn and returns any recovered panic value, or nil if no panic occurred.
func catchPanic(fn func()) (val interface{}) {
	defer func() { val = recover() }()
	fn()
	return nil
}
