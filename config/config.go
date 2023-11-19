// Package config provides configuration for different parts of Bidon services, that is shared between them
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var Env = getEnv()

const (
	ProdEnv = "production"
	DevEnv  = "development"
	TestEnv = "test"
)

func LoadEnvFile() {
	var err error
	if Env == TestEnv {
		err = godotenv.Load(".env.test")
	} else {
		err = godotenv.Load()
	}
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
}

func getEnv() string {
	switch env := os.Getenv("ENVIRONMENT"); env {
	case ProdEnv:
		return ProdEnv
	case TestEnv:
		return TestEnv
	default:
		return DevEnv
	}
}
