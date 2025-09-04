package handler

import (
	"github.com/haatos/simple-ci/internal/views/pages"
	"github.com/labstack/echo/v4"
)

func GetAppPage(c echo.Context) error {
	u := getCtxUser(c)
	if isHXRequest(c) {
		return render(c, pages.AppMain())
	}
	return render(c, pages.AppPage(u))
}
