package service

import (
	"blum-test/common/models"
	"errors"
	"fmt"
)

var ErrServiceStarted = errors.New("service is already started")
var ErrServiceInternal = errors.New("service internal error")
var ErrInvalidInternalRate = errors.New("invalid rate for pair, please try later")

type ErrRateIsNotAvailable struct {
	Code models.CurrencyCode
}

func (e *ErrRateIsNotAvailable) Error() string {
	return fmt.Sprintf("rate for \"%s\" is currently not available, please try later", e.Code)
}
