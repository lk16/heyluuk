package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lk16/heyluuk/internal/redirect"
)

var (
	postgresDB       = os.Getenv("POSTGRES_DB")
	postgresUser     = os.Getenv("POSTGRES_USER")
	postgresPassword = os.Getenv("POSTGRES_PASSWORD")
)

const postgresHost = "db"

func main() {

	dsn := fmt.Sprintf("host=%s sslmode=disable user=%s password=%s dbname=%s", postgresHost,
		postgresUser, postgresPassword, postgresDB)

	db, err := gorm.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err.Error())
	}

	errs := redirect.Migrate(db)
	if len(errs) > 0 {
		for _, err := range errs {
			log.Println(err.Error())
		}
		panic("Redirect DB migration failed")
	}

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	controller := &redirect.Controller{DB: db}
	e.GET("/*", controller.Redirect)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
