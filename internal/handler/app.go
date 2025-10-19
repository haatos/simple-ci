package handler

import (
	"net/http"
	"time"

	"github.com/haatos/simple-ci/internal"
	"github.com/haatos/simple-ci/internal/views"
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

func GetConfigPage(c echo.Context) error {
	u := getCtxUser(c)
	if isHXRequest(c) {
		return render(c, pages.ConfigMain(internal.Config))
	}
	return render(c, pages.ConfigPage(u, internal.Config))
}

func PostConfig(c echo.Context) error {
	cp := new(ConfigParams)
	if err := c.Bind(cp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid config data")
	}

	config := &internal.Configuration{
		SessionExpiresHours: internal.HoursDuration(
			time.Duration(cp.SessionExpiresHours) * time.Hour,
		),
		QueueSize: cp.QueueSize,
	}

	if err := internal.UpdateConfiguration(config); err != nil {
		return newError(
			c, err,
			http.StatusInternalServerError,
			"unable to update configuration file",
		)
	}

	return renderToast(c, views.SuccessToast("configuration updated", 3000))
}
