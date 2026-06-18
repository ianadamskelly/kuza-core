package config

import "os"

type Config struct {
	Env             string
	Addr            string
	DatabaseURL     string
	PublicURL       string
	StorageEndpoint string
	StorageBucket   string
	StorageAccess   string
	StorageSecret   string
}

func Load() Config {
	return Config{
		Env:             env("KUZA_CORE_ENV", "development"),
		Addr:            env("KUZA_CORE_ADDR", ":8080"),
		DatabaseURL:     env("KUZA_CORE_DATABASE_URL", ""),
		PublicURL:       env("KUZA_CORE_PUBLIC_URL", "http://localhost:8080"),
		StorageEndpoint: env("KUZA_CORE_STORAGE_ENDPOINT", ""),
		StorageBucket:   env("KUZA_CORE_STORAGE_BUCKET", "kuza-core"),
		StorageAccess:   env("KUZA_CORE_STORAGE_ACCESS_KEY", ""),
		StorageSecret:   env("KUZA_CORE_STORAGE_SECRET_KEY", ""),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
