package main

import (
	"github.com/bmerchant22/pkg/store"
	"github.com/bmerchant22/pkg/web"
	"go.uber.org/zap"
	"log"
)

func main() {
	var logger *zap.Logger
	var loggerErr error

	if logger, loggerErr = zap.NewDevelopment(); loggerErr != nil {
		log.Fatalln(loggerErr)
	}

	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	p := store.MongoStore{}

	p.ConnectToDatabase()

	web.CreateWebServer(&p)
}
