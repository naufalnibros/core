package middleware

import (
	"app/src/utils"
	"app/src/utils/env"
	"app/src/utils/logger"
	"app/src/utils/txid"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func OnReceive(context *fiber.Ctx) error {
	defer func(context *fiber.Ctx) {
		if iserrror := utils.RecoverContext(recover(), context, false); iserrror {
			HTTP200(context, "50", "Terjadi kesalahan pada server. Silakan coba beberapa saat lagi.", nil)
		}
	}(context)

	POD_ID, _ := strconv.Atoi(env.Get("POD_ID"))
	txID := txid.Next(POD_ID)
	context.Locals("txID", txID)
	context.Set("txID", txID)

	protocol := context.Protocol()
	hostname := context.Hostname()

	log := logger.New(context.Method()+": "+protocol+"://"+hostname+context.OriginalURL(), txID)
	context.Locals("logger", log)

	log.Info("Receiving: " + strings.ReplaceAll(string(context.BodyRaw()), "\n", ""))

	return context.Next()
}
