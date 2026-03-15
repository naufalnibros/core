package wallets

import (
	"app/src/exec"
	"app/src/middleware"
	models "app/src/models/wallets"
	"app/src/utils/currency"
	"app/src/utils/logger"

	"github.com/gofiber/fiber/v2"
)

func Create(context *fiber.Ctx) (err error) {
	log := context.Locals("logger").(*logger.CustomLogger)

	req := models.Request()
	defer req.Release()

	if errMsg := req.Bind(context); errMsg != "" {
		return middleware.HTTP200(context, "40", errMsg, nil)
	}

	if req.SourceAccountNo == "" || req.Currency == "" {
		return middleware.HTTP200(context, "40", "Invalid Format: Field 'sourceAccountNo' dan 'currency' wajib di isi.", nil)
	}

	if !currency.Cek(req.Currency) {
		return middleware.HTTP200(context, "40", "Invalid Currency: Kode mata uang '"+req.Currency+"' tidak valid (ISO 4217).", nil)
	}

	wallet := exec.Wallet(context.Context(), req.SourceAccountNo, req.Currency)
	if err := wallet.Create(); err != nil {
		log.Error(err, "Failed to create wallet")
		errMsg := exec.ErrWallet(context, err).Error()
		return middleware.HTTP200(context, "50", errMsg, nil, err)
	}

	return middleware.HTTP200(context, "00", "Wallet created successfully.", nil)
}
