package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/haatos/simple-ci/internal/views"
	"github.com/haatos/simple-ci/internal/views/pages"
	"github.com/labstack/echo/v4"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type RunCancelError struct {
	Message string
}

func (rce RunCancelError) Error() string {
	return rce.Message
}

func ErrorHandler(err error, c echo.Context) {
	switch e := err.(type) {
	case *echo.HTTPError:
		c.Logger().Errorf(
			"handler internal error %s [%d]: %+v\n",
			c.Request().URL.Path, e.Code, e.Internal,
		)
		errorPage(c, e.Code, e.Message.(string))
	case *HTMXError:
		c.Logger().Errorf(
			"handler internal error %s [%d]: %+v\n",
			c.Request().URL.Path, e.Code, e.Internal,
		)
		renderToast(c, views.FailureToast(e.Message.(string), 4000))
	default:
		c.Logger().Errorf("handler error: %+v\n", e)
		c.JSON(
			http.StatusInternalServerError,
			echo.HTTPError{Message: "something went terribly wrong"},
		)
	}
}

func errorPage(c echo.Context, status int, message string) error {
	u := getCtxUser(c)
	title := fmt.Sprintf("%d - %s", status, http.StatusText(status))
	if isHXRequest(c) {
		return render(c, pages.ErrorMain(title, message))
	}
	return render(c, pages.ErrorPage(u, title, message))
}

func isUniqueConstraintError(err error) bool {
	var sqErr *sqlite.Error
	if errors.As(err, &sqErr) {
		return sqErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
	}
	return false
}

func isForeignKeyConstraintError(err error) bool {
	var sqErr *sqlite.Error
	if errors.As(err, &sqErr) {
		return sqErr.Code() == sqlite3.SQLITE_CONSTRAINT_TRIGGER ||
			sqErr.Code() == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY
	}
	return false
}

func newError(c echo.Context, err error, status int, message string) error {
	if isHXRequest(c) {
		e := newHTMXError(status, message)
		if err != nil {
			e = e.WithInternal(err)
		}
		return e
	}

	e := echo.NewHTTPError(status, message)
	if err != nil {
		e = e.WithInternal(err)
	}
	return e
}

func newErrorString(c echo.Context, err error, status int, target, message string) error {
	hxReswap(c, "innerHTML")
	hxRetarget(c, target)
	c.Logger().Errorf(
		"handler internal error %s [%d]: %+v\n",
		c.Request().URL.Path, status, err,
	)
	return c.String(status, message)
}
