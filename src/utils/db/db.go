package db

import (
	"app/src/utils/env"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2/log"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

var (
	DB   NewConnection
	once sync.Once
)

type NewConnection struct {
	DbName          string
	DbSource        string
	MaxOpenConn     int
	MaxIdleConn     int
	MaxConnLifetime int
	Conn            *sqlx.DB
}

func Connect() {
	once.Do(func() {
		dbmain := env.Get("DB_MAIN")

		DB = createConnection(NewConnection{
			DbName:   "DB MAIN",
			DbSource: dbmain,

			MaxOpenConn:     20,
			MaxIdleConn:     20,
			MaxConnLifetime: 30,
		})
	})
}

func Conn() *sqlx.DB {
	return DB.Conn
}

func Close() {
	if DB.Conn != nil {
		DB.Conn.Close()
		log.Info("Database connection closed gracefully")
	}
}

func createConnection(connection NewConnection) NewConnection {
	db, err := sqlx.Open("pgx", connection.DbSource)
	if err != nil {
		log.Fatal("failed to initialize database pool [" + connection.DbName + "]: " + err.Error())
	}

	db.SetMaxOpenConns(connection.MaxOpenConn)
	db.SetMaxIdleConns(connection.MaxIdleConn)
	db.SetConnMaxLifetime(time.Duration(connection.MaxConnLifetime) * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal("failed to reach database [" + connection.DbName + "] on startup: " + err.Error())
	}

	connection.Conn = db
	log.Info("create connection pool [" + connection.DbName + "] success")

	return connection
}
