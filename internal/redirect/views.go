package redirect

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (cont *Controller) Index(c echo.Context) error {
	type dataType struct {
		CaptchaSiteKey string
	}

	data := dataType{CaptchaSiteKey: captchaSiteKey}
	return c.Render(http.StatusOK, "index.html", data)
}

func (cont *Controller) Faq(c echo.Context) error {
	type dataType struct {
		CaptchaSiteKey string
	}

	data := dataType{CaptchaSiteKey: captchaSiteKey}
	return c.Render(http.StatusOK, "faq.html", data)
}

func (cont *Controller) Predictions(c echo.Context) error {
	type dataType struct {
		CaptchaSiteKey string
	}

	data := dataType{CaptchaSiteKey: captchaSiteKey}
	return c.Render(http.StatusOK, "predictions.html", data)
}
