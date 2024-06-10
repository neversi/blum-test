package http

import (
	"blum-test/common/models"
	"blum-test/internal/service"
	"errors"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type ConvertResponse struct {
	Output float64 `json:"output"`
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Convert
// @Summary      Converts amount of base currency to quote currency
// @Description  Converts Fiat/Crypto and Crypto/Fiat currency pairs
// @Tags         rates
// @Produce      json
// @Param        base      query     string   true   "base currency code"             example(USD)
// @Param        quote     query     string   true   "quote currency code"            example(ETH)
// @Param        amount    query     number   true   "input amount of base currency"  example(100)
// @Param        decimals  query     integer  false  "round up to decimals places"    example(5)  default(5)
// @Success      200       {object}  ConvertResponse
// @Failure      400       {object}  ErrorResponse  "invalid parameters"
// @Failure      422       {object}  ErrorResponse  "currency or rate not exists"
// @Failure      500
// @Router       /convert [get]
func (s *Server) Convert(c *fiber.Ctx) error {
	base := c.Query("base")
	quote := c.Query("quote")
	amountStr := c.Query("amount")
	decimalsStr := c.Query("decimals")

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: err.Error(),
		})
	}

	var decimals int64 = 5

	if decimalsStr != "" {
		decimals, err = strconv.ParseInt(decimalsStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: err.Error(),
			})
		}
	}

	res, err := s.svc.Convert(c.Context(), base, quote, amount, decimals)
	if err != nil {
		if errors.Is(err, service.ErrServiceInternal) {
			return c.SendStatus(fiber.ErrInternalServerError.Code)
		}
		if errors.Is(err, service.ErrInvalidInternalRate) {
			return c.Status(http.StatusUnprocessableEntity).JSON(ErrorResponse{
				Error: err.Error(),
			})
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

	return c.Status(http.StatusOK).JSON(ConvertResponse{
		Output: res,
	})
}
