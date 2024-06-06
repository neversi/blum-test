package main

import (
	"blum-test/common/config"
	"blum-test/common/logger"
	"blum-test/common/models"
	"blum-test/internal/clients/fastforex"
	"blum-test/internal/db"
	"blum-test/internal/repository"
	"blum-test/internal/service"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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

	svc := service.NewRateCalculator(cfg.Service, repo, fastForexClient)
	go func() {
		if err := svc.Start(ctx); err != nil {
			logger.JSONLogger.Error("failed to start service", slog.Any("error", err))
			return
		}
	}()

	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		JSONEncoder:   json.Marshal,
	})

	app.Get("/convert", func(c *fiber.Ctx) error {
		base := c.Query("base")
		quote := c.Query("quote")
		amountStr := c.Query("amount")

		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: err.Error(),
			})
		}

		res, err := svc.Convert(c.Context(), base, quote, amount)
		if err != nil {
			if errors.Is(err, service.ErrServiceInternal) {
				return c.SendStatus(fiber.ErrInternalServerError.Code)
			}
			var currencyNotAvailable *models.ErrCurrencyNotAvailable
			if errors.As(err, &currencyNotAvailable) {
				return c.Status(http.StatusUnprocessableEntity).JSON(ErrorResponse{
					Error: err.Error(),
				})
			}
			var invalidCurrencyPair *models.ErrInvalidCurrencyPair
			if errors.As(err, &invalidCurrencyPair) {
				return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
					Error: err.Error(),
				})
			}

			return c.SendStatus(http.StatusInternalServerError)
		}

		return c.Status(http.StatusOK).JSON(fiber.Map{
			"output": res,
		})
	})

	go func() {
		if err := app.Listen(fmt.Sprintf("%s:%d", cfg.HTTPServer.Host, cfg.HTTPServer.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.JSONLogger.Error("error while listening http port", err)
		}
	}()
	<-ctx.Done()
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		logger.JSONLogger.Error("error while shutdowning http server", err)
	}
}
