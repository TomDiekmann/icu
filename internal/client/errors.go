package client

import "fmt"

const (
	ExitSuccess     = 0
	ExitGeneral     = 1
	ExitAuth        = 2
	ExitNotFound    = 3
	ExitValidation  = 4
	ExitRateLimited = 5
	ExitNetwork     = 6
)

type APIError struct {
	StatusCode int
	Body       string
	ExitCode   int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Body)
}

func newAPIError(statusCode int, body string) *APIError {
	exit := ExitGeneral
	switch {
	case statusCode == 401 || statusCode == 403:
		exit = ExitAuth
	case statusCode == 404:
		exit = ExitNotFound
	case statusCode == 422:
		exit = ExitValidation
	case statusCode == 429:
		exit = ExitRateLimited
	}
	return &APIError{StatusCode: statusCode, Body: body, ExitCode: exit}
}
