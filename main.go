package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/kaedwen/speedtest/cmd"
	"go.uber.org/zap"
)

func main() {
	lg, err := zap.NewDevelopment()
	if err != nil {
		os.Exit(1)
	}

	cmd := cmd.NewTestCommand(lg)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	// run the command
	if err := cmd.ExecuteContext(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(10)
		}

		os.Exit(1)
	}
}
