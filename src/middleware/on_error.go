package middleware

import (
	"app/src/utils/env"
	"app/src/utils/logger"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func OnError(context *fiber.Ctx, err error) error {
	mode := env.Get("DEPLOYMENT_MODE")

	txID := ""
	if v := context.Locals("txID"); v != nil {
		txID, _ = v.(string)
	}

	var errormsg *string
	if mode == "dev" {
		msg := err.Error()
		errormsg = &msg
	}

	if l := context.Locals("logger"); l != nil {
		log := l.(*logger.CustomLogger)
		log.Info("UnRecovered=", err.Error())
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code := fmt.Sprintf("%d", fiberErr.Code)
		message := "Terjadi kesalahan."

		switch fiberErr.Code {
		case fiber.StatusNotFound:
			message = "Route tidak ditemukan."
		case fiber.StatusMethodNotAllowed:
			message = "Method tidak diperbolehkan."
		default:
			message = fiberErr.Message
		}

		response := map[string]any{
			"attribute": map[string]any{
				"txID":     txID,
				"code":     code,
				"message":  message,
				"errormsg": errormsg,
			},
			"result": nil,
		}

		return context.Status(fiberErr.Code).JSON(response)
	}

	response := map[string]any{
		"attribute": map[string]any{
			"txID":     txID,
			"code":     "500",
			"message":  "Terjadi kesalahan pada server. Silakan coba beberapa saat lagi.",
			"errormsg": errormsg,
		},
		"result": nil,
	}

	return context.Status(fiber.StatusInternalServerError).JSON(response)
}
