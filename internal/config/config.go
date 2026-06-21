package config

import (
	"bufio"
	"os"
	"strings"
)

type AppConfig struct {
	Host         string
	Port         string
	MongoURI     string
	DatabaseName string
	JWTSecret    string
	UploadDir    string
}

var Values AppConfig

func Load() AppConfig {
	loadDotEnv(".env")

	Values = AppConfig{
		Host:         getEnv("HOST", "127.0.0.1"),
		Port:         getEnv("PORT", "8080"),
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DatabaseName: getEnv("MONGO_DATABASE", "bugyou"),
		JWTSecret:    getEnv("JWT_SECRET", "change-this-secret"),
		UploadDir:    getEnv("UPLOAD_DIR", "uploads"),
	}

	return Values
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}
