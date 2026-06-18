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
	SessionTTLHours int
	Bootstrap       BootstrapConfig
}

type BootstrapConfig struct {
	ProjectName   string
	ProjectSlug   string
	OwnerEmail    string
	OwnerPassword string
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
		SessionTTLHours: envInt("KUZA_CORE_SESSION_TTL_HOURS", 24),
		Bootstrap: BootstrapConfig{
			ProjectName:   env("KUZA_CORE_BOOTSTRAP_PROJECT_NAME", ""),
			ProjectSlug:   env("KUZA_CORE_BOOTSTRAP_PROJECT_SLUG", ""),
			OwnerEmail:    env("KUZA_CORE_BOOTSTRAP_OWNER_EMAIL", ""),
			OwnerPassword: env("KUZA_CORE_BOOTSTRAP_OWNER_PASSWORD", ""),
		},
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	var parsed int
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return fallback
		}
		parsed = parsed*10 + int(digit-'0')
	}
	if parsed == 0 {
		return fallback
	}
	return parsed
}
