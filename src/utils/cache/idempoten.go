package cache

import (
	"errors"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gofiber/fiber/v2/log"
)

const (
	IdempotencyTTL    = 10 // 10sec
	IdempotencyPrefix = "LOCK:"
)

var ErrDuplicateRequest = errors.New("duplicate transaction")

func AcquireLock(key string) error {
	if MC == nil {
		log.Warn("[IDEMPOTENCY] Memcached client nil, skipping lock")
		return nil
	}

	cacheKey := IdempotencyPrefix + key

	err := MC.Add(&memcache.Item{
		Key:        cacheKey,
		Value:      []byte("1"),
		Expiration: IdempotencyTTL,
	})

	if err == nil {
		return nil
	}

	if errors.Is(err, memcache.ErrNotStored) {
		return ErrDuplicateRequest
	}

	log.Warn("[IDEMPOTENCY] Memcached error (fallback allow): ", err)
	return nil
}

func ReleaseLock(key string) {
	if MC == nil {
		return
	}

	cacheKey := IdempotencyPrefix + key

	if err := MC.Delete(cacheKey); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		log.Warn("[IDEMPOTENCY] Failed to release lock: ", err)
	}
}
