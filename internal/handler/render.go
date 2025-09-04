package handler

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

func render(c echo.Context, component templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := component.Render(c.Request().Context(), buf); err != nil {
		return err
	}
	return c.HTML(http.StatusOK, buf.String())
}

func renderToast(c echo.Context, component templ.Component) error {
	c.Response().Header().Set("hx-push-url", "false")
	c.Response().Header().Set("hx-retarget", "body")
	c.Response().Header().Set("hx-reswap", "beforeend")
	return render(c, component)
}
