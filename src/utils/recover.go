package utils

import (
	"app/src/utils/logger"
	"runtime"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

func RecoverContext(recover any, context *fiber.Ctx, nextPanic ...bool) (isError bool) {
	log := context.Locals("logger").(*logger.CustomLogger)

	unrecover := true

	if len(nextPanic) > 0 {
		unrecover = nextPanic[0]
	}

	frame, file, line, _ := runtime.Caller(4)

	if recover != nil {
		if err, ok := recover.(error); ok {
			log.RecoverError(frame, file, line, err)
		} else {
			log.RecoverInfo(frame, file, line, recover, debug.Stack())
		}

		if unrecover {
			panic(recover)
		} else {
			return true

		}
	}

	return false
}
