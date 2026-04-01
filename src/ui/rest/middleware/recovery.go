package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

const rateLimitCooldown = 5 * time.Second

func Recovery() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var nextErr error
		panicVal := catchPanic(func() {
			nextErr = ctx.Next()
		})

		if panicVal == nil {
			return nextErr
		}

		errMsg := fmt.Sprintf("%v", panicVal)

		// WhatsApp rate limit (error 479/420): set device cooldown, wait 5s, retry once
		if isWhatsAppRateLimit(errMsg) {
			if device, ok := ctx.Locals("device").(*whatsapp.DeviceInstance); ok && device != nil {
				device.SetRateLimited(rateLimitCooldown)
			}

			retryCount, _ := ctx.Locals("__rate_limit_retry").(int)
			if retryCount < 1 {
				ctx.Locals("__rate_limit_retry", retryCount+1)
				deviceID, _ := ctx.Locals("device_id").(string)
				logrus.Warnf("[RATE_LIMIT] WhatsApp rate limit hit for device %s, retrying in %s...", deviceID, rateLimitCooldown)
				time.Sleep(rateLimitCooldown)
				return ctx.RestartRouting()
			}

			// Retry also failed — return 429
			deviceID, _ := ctx.Locals("device_id").(string)
			logrus.Errorf("[RATE_LIMIT] WhatsApp rate limit persists for device %s after retry", deviceID)
			ctx.Set("Retry-After", fmt.Sprintf("%.0f", rateLimitCooldown.Seconds()))
			return ctx.Status(fiber.StatusTooManyRequests).JSON(utils.ResponseData{
				Status:  fiber.StatusTooManyRequests,
				Code:    "RATE_LIMITED",
				Message: fmt.Sprintf("WhatsApp rate limit hit, retry after %s", rateLimitCooldown),
			})
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

func isWhatsAppRateLimit(errMsg string) bool {
	return strings.Contains(errMsg, "server returned error 479") ||
		strings.Contains(errMsg, "server returned error 420")
}
