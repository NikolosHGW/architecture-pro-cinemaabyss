package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("events service stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.kafkaBrokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordDeliveryTimeout(publishTimeout),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeTopics(movieTopic, userTopic, paymentTopic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		return err
	}
	defer client.CloseAllowingRebalance()

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.port),
		Handler:           newHandler(client),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- consumeEvents(ctx, client, slog.Default())
		cancel()
	}()
	shutdownDone := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		shutdownDone <- server.Shutdown(shutdownCtx)
	}()

	slog.Info("events HTTP server started", "port", cfg.port)
	serveErr := server.ListenAndServe()
	cancel()
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, <-consumerDone, <-shutdownDone)
}
