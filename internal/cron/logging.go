package cron

import (
	"os"

	"github.com/sirupsen/logrus"
)

func SetupLogging() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	logLevel, err := logrus.ParseLevel(getEnvOrDefault("LOG_LEVEL", "info"))
	if err != nil {
		log.Warnf("failed to parse log level: %s", err)
		logLevel = logrus.InfoLevel
	}
	log.SetLevel(logLevel)
	return log
}

func getEnvOrDefault(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
