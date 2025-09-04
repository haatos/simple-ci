package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type HTMXError struct {
	Internal error `json:"-"` // Stores the error returned by an external dependency
	Message  any   `json:"message"`
	Code     int   `json:"-"`
}

func newHTMXError(code int, message ...any) *HTMXError {
	he := &HTMXError{Code: code, Message: http.StatusText(code)}
	if len(message) > 0 {
		he.Message = message[0]
	}
	return he
}

// Error makes it compatible with `error` interface.
func (he *HTMXError) Error() string {
	if he.Internal == nil {
		return fmt.Sprintf("code=%d, message=%v", he.Code, he.Message)
	}
	return fmt.Sprintf("code=%d, message=%v, internal=%v", he.Code, he.Message, he.Internal)
}

// SetInternal sets error to HTMXError.Internal
func (he *HTMXError) SetInternal(err error) *HTMXError {
	he.Internal = err
	return he
}

// WithInternal returns clone of HTMXError with err set to HTMXError.Internal field
func (he *HTMXError) WithInternal(err error) *HTMXError {
	return &HTMXError{
		Code:     he.Code,
		Message:  he.Message,
		Internal: err,
	}
}

// Unwrap satisfies the Go 1.13 error wrapper interface.
func (he *HTMXError) Unwrap() error {
	return he.Internal
}

func isHXRequest(c echo.Context) bool {
	return c.Request().Header.Get("hx-request") != ""
}

func hxRedirect(c echo.Context, url string) error {
	c.Response().Header().Set("hx-redirect", url)
	return nil
}

func hxRetarget(c echo.Context, target string) error {
	c.Response().Header().Set("hx-retarget", target)
	return nil
}

func hxReswap(c echo.Context, swap string) error {
	c.Response().Header().Set("hx-reswap", swap)
	return nil
}
