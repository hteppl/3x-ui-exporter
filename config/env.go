package config

import (
	"os"

	"github.com/joho/godotenv"
)

// DefaultEnvFile is used when ENV_FILE is not set.
const DefaultEnvFile = ".env"

// loadEnvFile loads ENV_FILE (default .env) into the environment. Existing
// variables win, so Docker and systemd stay authoritative. A missing default
// file is not an error; a missing explicit ENV_FILE is.
func loadEnvFile() error {
	path := os.Getenv("ENV_FILE")
	explicit := path != ""
	if !explicit {
		path = DefaultEnvFile
	}

	if err := godotenv.Load(path); err != nil {
		if !explicit && os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}
