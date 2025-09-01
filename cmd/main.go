package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaedwen/speedtest/pkg/app"
	"go.uber.org/zap"
)

func main() {
	log, err := zap.NewDevelopment()
	if err != nil {
		os.Exit(1)
	}

	app := app.NewApplication(log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.Run(ctx); err != nil {
		log.Fatal("failed to run application", zap.Error(err))
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs
	log.Info("Received signal", zap.String("signal", sig.String()))

	cancel()
	app.Close()
	log.Info("Application stopped")
}
