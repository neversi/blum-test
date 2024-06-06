package fastforex

import "errors"

var ErrRateLimit = errors.New("rate limit exceeded")
var ErrInvalidAPIKey = errors.New("invalid API Key")
