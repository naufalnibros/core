package main

import (
	"app/src/middleware"
	"app/src/routes"
	"app/src/utils/cache"
	"app/src/utils/db"
	"app/src/utils/env"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goccy/go-json"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	/** DEFINE: INIT ENVs **/
	env.Init()

	/** DEFINE: DB Conn **/
	db.Connect()
	defer db.Close()

	/** DEFINE: Memcached Conn **/
	cache.Connect()
	defer cache.Close()

	/** INIT App Fiber **/
	app := fiber.New(fiber.Config{
		AppName:               env.Get("APP_NAME") + " " + env.Get("APP_VERSION"),
		Concurrency:           256 * 1024,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
		DisableStartupMessage: true,
		ErrorHandler:          middleware.OnError,
	})

	app.Use(fiberrecover.New())
	app.Use(favicon.New())
	app.Use(middleware.OnReceive)

	app.Get("/", routes.Homepage)
	app.Get("/_health", routes.Health)

	apppath := app.Group(env.Get("APP_PATH"))
	apppath.Get("/", routes.Homepage)
	apppath.Get("/_health", routes.Health)

	apppath.Post("/:processor", routes.Service)
	apppath.Post("/:processor/:method", routes.Service)

	/** GRACEFUL SHUTDOWN HANDLER **/
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdown
		log.Info("Gracefully shutting down...")

		if err := app.ShutdownWithTimeout(15 * time.Second); err != nil {
			log.Errorf("Forcefully shutting down: %v", err)
		}
	}()

	port := env.Get("APP_PORT")
	if len(port) > 0 && port[0] != ':' {
		port = ":" + port
	}

	log.Infof("Start Service on http://localhost%s%s", port, env.Get("APP_PATH"))

	if err := app.Listen(port); err != nil {
		log.Panic("Start error service:", err)
	}

	log.Info("Stopped Service...")
}
