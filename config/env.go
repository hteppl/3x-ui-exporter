package config

import (
	"os"

	"github.com/joho/godotenv"
)

// DefaultEnvFile is the env file loaded from the working directory when
// ENV_FILE is not set.
const DefaultEnvFile = ".env"

// loadEnvFile loads key/value pairs from an env file into the process
// environment so that they are visible to the flag parser.
//
// The path is taken from ENV_FILE, falling back to .env in the working
// directory. Values already present in the environment win, which keeps
// container and systemd variables authoritative over the file.
//
// A missing file is not an error: the exporter is fully configurable through
// environment variables and command-line flags alone.
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
