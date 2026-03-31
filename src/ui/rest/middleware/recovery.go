package middleware

import (
	"context"
	"fmt"
	"strings"

	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func Recovery() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		defer func() {
			err := recover()
			if err != nil {
				var res utils.ResponseData
				res.Status = 500
				res.Code = "INTERNAL_SERVER_ERROR"
				res.Message = fmt.Sprintf("%v", err)

				// Check for typed errors first
				if ctxErr, ok := err.(error); ok && ctxErr == context.DeadlineExceeded {
					res.Status = 504
					res.Code = "GATEWAY_TIMEOUT"
					res.Message = "Request timed out waiting for WhatsApp server response"
				}

				if errGeneric, ok := err.(pkgError.GenericError); ok {
					res.Status = errGeneric.StatusCode()
					res.Code = errGeneric.ErrCode()
					res.Message = errGeneric.Error()
				} else if strings.Contains(strings.ToLower(res.Message), "not found") {
					res.Status = 404
					res.Code = "NOT_FOUND"
				}

				if res.Status >= 500 {
					logrus.Errorf("Panic recovered in middleware: %v", err)
				} else {
					logrus.Warnf("Recovered in middleware: %v", err)
				}

				_ = ctx.Status(res.Status).JSON(res)
			}
		}()

		return ctx.Next()
	}
}
