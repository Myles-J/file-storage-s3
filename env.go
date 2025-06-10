package main

import (
	"log"
	"os"
)

// MustGetenv is a helper function to get environment variables and panic if they are not set
func MustGetenv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is not set", key)
	}
	return value
}
