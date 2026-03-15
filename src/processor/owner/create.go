package owner

import (
	"app/src/exec"
	"app/src/middleware"

	"github.com/gofiber/fiber/v2"
)

func Create(context *fiber.Ctx) (err error) {

	ID, err := exec.Owner(context.Context()).Create()
	if err != nil {
		return middleware.HTTP200(context, "01", "Failed to create owner.", nil)
	}

	result := map[string]any{"owner_id": ID}

	return middleware.HTTP200(context, "00", "Owner created successfully.", result)
}
