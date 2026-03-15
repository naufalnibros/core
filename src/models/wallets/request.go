package wallets

import (
	"app/src/utils/logger"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
)

type WalletRequest struct {
	SourceAccountNo     string `json:"sourceAccountNo" form:"sourceAccountNo"`
	BeneﬁciaryAccountNo string `json:"beneﬁciaryAccountNo" form:"beneﬁciaryAccountNo"`
	Currency            string `json:"currency" form:"currency"`
	Amount              string `json:"amount" form:"amount"`
}

var walletReqPool = sync.Pool{
	New: func() any { return new(WalletRequest) },
}

func Request() *WalletRequest {
	return walletReqPool.Get().(*WalletRequest)
}

func (r *WalletRequest) Release() {
	*r = WalletRequest{}
	walletReqPool.Put(r)
}

func (r *WalletRequest) Bind(context *fiber.Ctx) string {
	if err := context.BodyParser(r); err != nil {
		log := context.Locals("logger").(*logger.CustomLogger)
		log.Error(err, "Failed to parse request body")
		return "Invalid Request: Format request tidak valid."
	}

	return ""
}

func (r *WalletRequest) ParseAmount(context *fiber.Ctx) (decimal.Decimal, string) {
	amount, err := decimal.NewFromString(r.Amount)
	if err != nil {
		log := context.Locals("logger").(*logger.CustomLogger)
		log.Error(err, "Invalid amount format: "+r.Amount)
		return decimal.Zero, "Invalid Amount: Format amount tidak valid."
	}

	return amount, ""
}
