package wallets

import (
	"app/src/exec"
	"app/src/middleware"
	models "app/src/models/wallets"
	"app/src/utils/logger"

	"github.com/gofiber/fiber/v2"
)

func Query(context *fiber.Ctx) (err error) {
	log := context.Locals("logger").(*logger.CustomLogger)

	req := models.Request()
	defer req.Release()

	if errMsg := req.Bind(context); errMsg != "" {
		return middleware.HTTP200(context, "40", errMsg, nil)
	}

	if req.SourceAccountNo == "" {
		return middleware.HTTP200(context, "40", "Invalid Format: Field 'sourceAccountNo' wajib diisi.", nil)
	}

	wallet := exec.Wallet(context.Context(), req.SourceAccountNo, "")
	wallets, err := wallet.Status()
	if err != nil {
		log.Error(err, "Failed to query wallets")
		errMsg := exec.ErrWallet(context, err).Error()
		return middleware.HTTP200(context, "50", errMsg, nil, err)
	}

	result := map[string]any{
		"owner_id": req.SourceAccountNo,
		"wallets":  wallets,
	}

	return middleware.HTTP200(context, "00", "Query wallet berhasil.", result)
}
