package main

import (
	"os"
	"strconv"
)

func GetEnvWithDefaultString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func GetEnvWithDefaultInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			return defaultVal
		}
		return v
	}

	return defaultVal
}
