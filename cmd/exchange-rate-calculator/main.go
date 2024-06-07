package main

import (
	"blum-test/common/apprunner"
	"blum-test/common/config"
	"blum-test/common/logger"
	"blum-test/internal/clients/fastforex"
	"blum-test/internal/db"
	deliveryHttp "blum-test/internal/delivery/http"
	"blum-test/internal/repository"
	"blum-test/internal/service"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
)

// @title           Rate Calculator API
// @version         0.1
// @description     Currencies rate calculator API with convertion functionality.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Abdarrakhman Akhmetgali
// @contact.email  neversi123123@gmail.com

// @BasePath  /v0
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		logger.JSONLogger.Error("parse config", err)
		return
	}

	logger.InitLogger(cfg.Name, cfg.LogLevel)

	dbClient, err := db.NewPostgresClient(ctx, cfg.Postgres)
	if err != nil {
		logger.JSONLogger.Error("initialize postgres client", err)
		return
	}
	defer dbClient.Close()

	repo := repository.NewCurrencyPostgresRepository(dbClient)

	fastForexClient, err := fastforex.NewClient(cfg.FastForex)
	if err != nil {
		logger.JSONLogger.Error("initialize fast forex client", err)
		return
	}

	// TODO shutdown after httpServer, maybe DI? or cascade shutdown
	svc := service.NewRateCalculator(cfg.Service, repo, fastForexClient)

	httpServer := deliveryHttp.NewServer(cfg, svc)

	if err := apprunner.StartApp(
		ctx,
		apprunner.NewRunner("rate calculator service", svc),
		apprunner.NewRunner("http server", httpServer),
	); err != nil && !errors.Is(err, context.Canceled) {
		logger.JSONLogger.Error("error running app", slog.Any("error", err))
	}
	logger.JSONLogger.Info("app finished")
}
