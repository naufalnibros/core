package cache

import (
	"app/src/utils/env"
	"sync"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gofiber/fiber/v2/log"
)

var (
	MC   *memcache.Client
	once sync.Once
)

func Connect() {
	once.Do(func() {
		host := env.Get("MEMCACHED_HOST")
		if host == "" {
			host = "memcached:11211"
		}

		MC = memcache.New(host)
		MC.MaxIdleConns = 10
		MC.Timeout = 100 * 1e6 // 100ms (time.Duration in nanoseconds)

		if err := MC.Ping(); err != nil {
			log.Warn("Memcached ping failed on startup (will retry on demand): ", err)
		} else {
			log.Info("Memcached connection [" + host + "] success")
		}
	})
}

func Close() {
	if MC != nil {
		MC.Close()
		log.Info("Memcached connection closed")
	}
}

func IsAvailable() bool {
	return MC != nil
}
