package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Handler
func hey(c echo.Context) error {
	return c.String(http.StatusOK, "Hey Luuk!")
}

func main() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", hey)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
