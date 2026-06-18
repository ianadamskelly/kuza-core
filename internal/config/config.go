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
	Bootstrap       BootstrapConfig
}

type BootstrapConfig struct {
	OrganizationName string
	OrganizationSlug string
	OwnerEmail       string
	OwnerPassword    string
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
		Bootstrap: BootstrapConfig{
			OrganizationName: env("KUZA_CORE_BOOTSTRAP_ORG_NAME", ""),
			OrganizationSlug: env("KUZA_CORE_BOOTSTRAP_ORG_SLUG", ""),
			OwnerEmail:       env("KUZA_CORE_BOOTSTRAP_OWNER_EMAIL", ""),
			OwnerPassword:    env("KUZA_CORE_BOOTSTRAP_OWNER_PASSWORD", ""),
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
