package tests

import (
	"app/src/middleware"
	"app/src/utils/db"
	"app/src/utils/env"
	"app/src/utils/logger"
	"app/src/utils/txid"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func NewTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.OnError,
	})

	app.Use(func(c *fiber.Ctx) error {
		txID := txid.Next(15)
		c.Locals("txID", txID)
		c.Set("txID", txID)

		log := logger.New(c.Method()+": "+c.OriginalURL(), txID)
		c.Locals("logger", log)

		return c.Next()
	})

	return app
}

var once sync.Once

func NewTestAppWithDB() *fiber.App {
	once.Do(func() {
		env.Init()

		dsn := env.Get("DB_TEST")

		conn, err := sqlx.Open("pgx", dsn)
		if err != nil {
			panic("test db open: " + err.Error())
		}

		if err := conn.Ping(); err != nil {
			panic("test db ping: " + err.Error())
		}

		conn.SetMaxOpenConns(5)
		conn.SetMaxIdleConns(2)
		conn.SetConnMaxLifetime(30 * time.Minute)

		db.DB = db.NewConnection{
			DbName:          "DB TEST",
			DbSource:        dsn,
			MaxOpenConn:     20,
			MaxIdleConn:     20,
			MaxConnLifetime: 30,
			Conn:            conn,
		}
	})

	return NewTestApp()
}

type ResponseBody struct {
	HTTPStatus int `json:"-"`
	Attribute  struct {
		TxID     string      `json:"txID"`
		Code     string      `json:"code"`
		Message  string      `json:"message"`
		ErrorMsg interface{} `json:"errormsg"`
	} `json:"attribute"`
	Result interface{} `json:"result"`
}
