package processor

import (
	"app/src/processor/owner"
	"app/src/processor/wallets"

	"github.com/gofiber/fiber/v2"
)

type HandlerFunc func(context *fiber.Ctx) error

type EndpointInfo struct {
	Handler     HandlerFunc
	Description string
}

// Format: Processor["processor"]["method"] = EndpointInfo
var Processor = map[string]map[string]EndpointInfo{
	"wallets": {
		"":         {wallets.Create, "Membuat wallet baru."},
		"topup":    {wallets.Topup, "Top up saldo wallet."},
		"payment":  {wallets.Payment, "Melakukan pembayaran."},
		"transfer": {wallets.Transfer, "Transfer saldo antar wallet."},
		"query":    {wallets.Query, "Cek saldo dan informasi wallet."},
	},
	"owner": {
		"": {owner.Create, "Registrasi owner."},
	},
}

func Handler(processor, method string) (HandlerFunc, bool) {
	if methods, ok := Processor[processor]; ok {
		if info, ok := methods[method]; ok {
			return info.Handler, true
		}
	}
	return nil, false
}
