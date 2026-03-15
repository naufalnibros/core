package wallets

import (
	"app/src/exec"
	"app/src/middleware"
	models "app/src/models/wallets"
	"app/src/utils/cache"
	"app/src/utils/currency"
	"app/src/utils/logger"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func Topup(context *fiber.Ctx) (err error) {
	log := context.Locals("logger").(*logger.CustomLogger)
	txID := context.Locals("txID").(string)

	req := models.Request()
	defer req.Release()

	if errMsg := req.Bind(context); errMsg != "" {
		return middleware.HTTP200(context, "40", errMsg, nil)
	}

	if req.SourceAccountNo == "" || req.Currency == "" || req.Amount == "" {
		return middleware.HTTP200(context, "40", "Invalid Format: Field 'sourceAccountNo', 'currency', dan 'amount' wajib diisi.", nil)
	}

	if !currency.Cek(req.Currency) {
		return middleware.HTTP200(context, "40", "Invalid Currency: Kode mata uang '"+req.Currency+"' tidak valid (ISO 4217).", nil)
	}

	amount, errMsg := req.ParseAmount(context)
	if errMsg != "" {
		return middleware.HTTP200(context, "40", errMsg, nil)
	}

	DUPLICATE_KEY := "TOPUP:" + req.SourceAccountNo + ":" + amount.String() + ":" + req.Currency
	if err := cache.AcquireLock(DUPLICATE_KEY); err != nil {
		if errors.Is(err, cache.ErrDuplicateRequest) {
			log.Info("Duplicate topup blocked: " + DUPLICATE_KEY)
			return middleware.Response(context, fiber.StatusTooEarly, "83", "Duplicate transaction.", nil)
		}
	}

	wallet := exec.Wallet(context.Context(), req.SourceAccountNo, req.Currency, txID)
	if err := wallet.Topup(amount, fmt.Sprintf("TOPUP SALDO %s %s", amount.String(), req.Currency)); err != nil {
		cache.ReleaseLock(DUPLICATE_KEY)

		log.Error(err, "Failed to topup")
		errMsg := exec.ErrWallet(context, err).Error()
		return middleware.HTTP200(context, "50", errMsg, nil, err)
	}

	return middleware.HTTP200(context, "00", fmt.Sprintf("Topup Successfull. Saldo berhasil ditambahkan sebesar %s %s", amount.String(), req.Currency), nil)
}
