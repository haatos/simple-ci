package handler

import (
	"github.com/haatos/simple-ci/internal/views/pages"
	"github.com/labstack/echo/v4"
)

func SetupAppRoutes(g *echo.Group) {
	appGroup := g.Group("/app", IsAuthenticated)
	appGroup.GET("", GetAppPage)
}

func GetAppPage(c echo.Context) error {
	u := getCtxUser(c)
	if isHXRequest(c) {
		return render(c, pages.AppMain(u))
	}
	return render(c, pages.AppPage(u))
}
