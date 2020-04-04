package internal

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lk16/heyluuk/internal/redirect"
	"gopkg.in/romanyx/recaptcha.v1"

	_ "github.com/jinzhu/gorm/dialects/postgres" // db driver
)

// TODO pass environment as dict to GetServer
var (
	captchaSiteKey   = os.Getenv("CAPTCHA_SITE_KEY")
	captchaSecretKey = os.Getenv("CAPTCHA_SECRET_KEY")
	postgresDB       = os.Getenv("POSTGRES_DB")
	postgresUser     = os.Getenv("POSTGRES_USER")
	postgresPassword = os.Getenv("POSTGRES_PASSWORD")
	postgresHost     = "db"
)

// GetServer returns a configured server
func GetServer() *echo.Echo {

	dsn := fmt.Sprintf("host=%s sslmode=disable user=%s password=%s dbname=%s", postgresHost,
		postgresUser, postgresPassword, postgresDB)

	db, err := gorm.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err.Error())
	}

	if err := redirect.Migrate(db); err != nil {
		panic("Redirect DB migration failed: " + err.Error())
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Renderer = NewTemplateRenderer()

	controller := &redirect.Controller{
		DB:      db,
		Captcha: recaptcha.New(captchaSecretKey)}

	e.GET("/*", controller.Redirect)

	e.Static("/static", "./web/static")
	e.Static("/static/jquery", "/npm/node_modules/jquery/dist")
	e.Static("/static/bootstrap", "/npm/node_modules/bootstrap/dist")
	e.Static("/static/patternfly-bootstrap-treeview", "/npm/node_modules/patternfly-bootstrap-treeview/dist")

	e.GET("/", redirectView("/at/my/site"))
	e.GET("/at/my/site", renderTemplateView("index.html"))
	e.GET("/at/my/faq", renderTemplateView("faq.html"))
	e.GET("/at/my/predictions", renderTemplateView("predictions.html"))
	e.GET("/at/this", renderTemplateView("new_link.html"))

	e.POST("/api/link", controller.PostLink)
	e.GET("/api/node/:id", controller.GetNode)
	e.GET("/api/node/:id/children", controller.GetNodeChildren)
	e.GET("/api/node/root", controller.GetNodeRoot)

	return e
}

func redirectView(URL string) func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.Redirect(http.StatusFound, URL)
	}
}
