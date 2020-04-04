package main

import (
	"github.com/lk16/heyluuk/internal"
)

const postgresHost = "db"

func main() {

	// Echo instance
	server := internal.GetServer()
	server.Logger.Fatal(server.Start(":8080"))
}
