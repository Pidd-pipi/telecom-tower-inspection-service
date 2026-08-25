package config

import (
	"os"
	"strconv"
)

func Port() string {
	value := os.Getenv("PORT")
	port, err := strconv.Atoi(value)
	if err == nil && port > 0 && port < 65536 {
		return value
	}
	return "8080"
}
