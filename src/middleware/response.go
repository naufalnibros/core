package middleware

import (
	"app/src/utils/env"
	"app/src/utils/logger"
	"fmt"
	"strings"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

func Response(context *fiber.Ctx, httpStatus int, code, message string, result any, errs ...error) error {
	log := context.Locals("logger").(*logger.CustomLogger)
	txID := context.Locals("txID").(string)

	mode := env.Get("DEPLOYMENT_MODE")

	errorMsg := ""
	if len(errs) > 0 && errs[0] != nil {
		errorMsg = errs[0].Error()
	}

	var errorMsg_ *string
	if mode == "dev" {
		msg := errorMsg
		errorMsg_ = &msg
	}

	response := map[string]any{
		"attribute": map[string]any{
			"txID":     txID,
			"code":     code,
			"message":  message,
			"errormsg": errorMsg_,
		},
		"result": result,
	}

	json, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Error converting to JSON:", err)
	}

	log.Info("Responding: " + strings.ReplaceAll(string(json), "\n", ""))

	return context.Status(httpStatus).JSON(response)
}

func HTTP200(context *fiber.Ctx, code, message string, result any, errs ...error) error {
	return Response(context, fiber.StatusOK, code, message, result, errs...)
}

func HTTP404(context *fiber.Ctx, code, message string, result any) error {
	return Response(context, fiber.StatusNotFound, code, message, result)
}

func HTTP504(context *fiber.Ctx, code, message string, result any) error {
	return Response(context, fiber.StatusGatewayTimeout, code, message, result)
}
