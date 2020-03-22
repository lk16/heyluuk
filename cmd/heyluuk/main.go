package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lk16/heyluuk/internal/redirect"
	"gopkg.in/romanyx/recaptcha.v1"
)

var (
	captchaSecretKey = os.Getenv("CAPTCHA_SECRET_KEY")
	postgresDB       = os.Getenv("POSTGRES_DB")
	postgresUser     = os.Getenv("POSTGRES_USER")
	postgresPassword = os.Getenv("POSTGRES_PASSWORD")
)

const postgresHost = "db"

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {

	dsn := fmt.Sprintf("host=%s sslmode=disable user=%s password=%s dbname=%s", postgresHost,
		postgresUser, postgresPassword, postgresDB)

	db, err := gorm.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err.Error())
	}

	if err := redirect.Migrate(db); err != nil {
		panic("Redirect DB migration failed: " + err.Error())
	}

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())

	e.Use(middleware.Recover())

	e.Renderer = &Template{
		templates: template.Must(template.ParseGlob("./web/template/*.html")),
	}

	controller := &redirect.Controller{
		DB:      db,
		Captcha: recaptcha.New(captchaSecretKey)}

	e.GET("/*", controller.Redirect)
	e.GET("/at/this", controller.NewLinkGet)

	e.POST("/api/link", controller.PostLink)
	e.GET("/api/node/:id", controller.GetNode)
	e.GET("/api/node/root", controller.GetNodeRoot)
	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
