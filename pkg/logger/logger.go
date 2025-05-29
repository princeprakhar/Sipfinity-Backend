package logger

import (
	"os"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func Init() {
	log = logrus.New()
	log.SetOutput(os.Stdout)
	
	if os.Getenv("ENVIRONMENT") == "production" {
		log.SetFormatter(&logrus.JSONFormatter{})
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		log.SetLevel(logrus.DebugLevel)
	}
}

func Info(args ...interface{}) {
	log.Info(args...)
}

func Error(args ...interface{}) {
	log.Error(args...)
}

func Debug(args ...interface{}) {
	log.Debug(args...)
}

func Warn(args ...interface{}) {
	log.Warn(args...)
}

func Fatal(args ...interface{}) {
	log.Fatal(args...)
}