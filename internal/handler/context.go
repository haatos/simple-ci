package handler

import (
	"github.com/haatos/simple-ci/internal/store"
	"github.com/labstack/echo/v4"
)

func getCtxUser(c echo.Context) *store.User {
	if u, ok := c.Get("user").(*store.User); ok {
		return u
	}
	return nil
}
