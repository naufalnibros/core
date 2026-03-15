package routes

import (
	"app/src/middleware"
	"app/src/processor"
	"app/src/utils/env"
	"fmt"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func Service(context *fiber.Ctx) (err error) {
	method := context.Params("method")
	processorName := context.Params("processor")

	handler, exists := processor.Handler(processorName, method)
	if !exists {
		return middleware.HTTP404(context, "40", "Processor atau method tidak ditemukan: "+processorName+"/"+method, nil)
	}

	return handler(context)
}

func Health(context *fiber.Ctx) (err error) {
	return context.Status(fiber.StatusOK).SendString("I'm Ok")
}

func Homepage(context *fiber.Ctx) error {
	appName := env.Get("APP_NAME")
	appVersion := env.Get("APP_VERSION")
	appPath := env.Get("APP_PATH")

	var endpointRows strings.Builder
	processorNames := make([]string, 0, len(processor.Processor))
	for name := range processor.Processor {
		processorNames = append(processorNames, name)
	}
	sort.Strings(processorNames)

	for _, procName := range processorNames {
		methods := processor.Processor[procName]
		methodNames := make([]string, 0, len(methods))
		for m := range methods {
			methodNames = append(methodNames, m)
		}
		sort.Strings(methodNames)

		for _, method := range methodNames {
			info := methods[method]
			var endpoint, label, methodLabel string
			if method == "" {
				endpoint = appPath + "/" + procName
				label = "POST " + endpoint
				methodLabel = "-"
			} else {
				endpoint = appPath + "/" + procName + "/" + method
				label = "POST " + endpoint
				methodLabel = method
			}
			endpointRows.WriteString(fmt.Sprintf(
				`<tr><td><code>%s</code></td><td><code>%s</code></td><td><code>%s</code></td><td>%s</td></tr>`,
				procName, methodLabel, label, info.Description,
			))
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; margin: 2rem; color: #222; }
h1 { font-size: 1.25rem; font-weight: 600; margin-bottom: 0.15rem; }
p { color: #666; font-size: 0.8rem; margin-bottom: 1.5rem; }
h2 { font-size: 0.85rem; font-weight: 600; color: #444; margin-bottom: 0.5rem; }
table { border-collapse: collapse; font-size: 0.8rem; }
th, td { text-align: left; padding: 6px 14px 6px 0; border-bottom: 1px solid #eee; }
th { color: #999; font-weight: 500; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.04em; }
code { font-family: 'SF Mono', Menlo, monospace; font-size: 0.78rem; }
a { color: #222; }
hr { border: none; border-top: 1px solid #eee; margin: 1.5rem 0; }
.muted { color: #999; font-size: 0.7rem; }
</style>
</head>
<body>
<h1>%s</h1>
<p>%s &middot; <a href="%s/_health">Health check</a></p>

<h2>Endpoints</h2>
<table>
<thead><tr><th>Processor</th><th>Method</th><th>Endpoint</th><th>Keterangan</th></tr></thead>
<tbody>%s</tbody>
</table>

<hr>
<span class="muted">%s %s</span>
</body>
</html>`, appName, appName, appVersion, appPath, endpointRows.String(), appName, appVersion)

	context.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return context.Status(fiber.StatusOK).SendString(html)
}
