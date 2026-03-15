package env

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2/log"
	"github.com/joho/godotenv"
)

var envs = map[string]string{
	"APP_NAME":        "Core Service",
	"APP_VERSION":     "1.0.0",
	"APP_PORT":        ":11082",
	"APP_PATH":        "/core/",
	"POD_ID":          "1",
	"APM_HOST":        "",
	"DEPLOYMENT_MODE": "dev",
	"DB_MAIN":         "",
	"DB_TEST":         "",
	"MEMCACHED_HOST":  "localhost:11211",
}

func Init() {
	err := godotenv.Load()

	if err != nil {
		dir, _ := os.Getwd()

		for i := 0; i < 3; i++ {
			dir = filepath.Dir(dir)
			envPath := filepath.Join(dir, ".env")
			if godotenv.Load(envPath) == nil {
				err = nil
				break
			}
		}
	}

	environ := os.Environ()
	var labels strings.Builder

	for _, env := range environ {
		split := strings.SplitN(env, "=", 2)
		if len(split) == 2 {
			key := split[0]
			value := split[1]

			if _, ok := envs[key]; ok {
				labels.WriteString("SETKEY: " + key + "\n")
				envs[key] = value
			}
		}
	}

	labels.WriteString("\n")

	if err == nil {
		log.Info("Success load .env file\n", labels.String())
	} else {
		log.Warn("No .env file found, relying purely on OS environment variables\n", labels.String())
	}
}

func Get(keyname string, defaultval ...string) string {

	env := os.Getenv(keyname)
	if env != "" {
		return env
	}

	if value, ok := envs[keyname]; ok && value != "" {
		return value
	}

	if len(defaultval) > 0 {
		if len(strings.TrimSpace(defaultval[0])) > 0 {
			return defaultval[0]
		}
	}

	return ""
}
