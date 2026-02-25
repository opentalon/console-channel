package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	consolechannel "github.com/opentalon/console-channel"
	"github.com/opentalon/opentalon/pkg/channel"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ch := consolechannel.New()
	if err := channel.Serve(ctx, ch); err != nil {
		log.Fatalf("console-channel: %v", err)
	}
}
