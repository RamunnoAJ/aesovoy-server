package main

import (
	"flag"
	"fmt"
	"github.com/RamunnoAJ/aesovoy-server/internal/app"
	"github.com/RamunnoAJ/aesovoy-server/internal/routes"
	"net/http"
	"time"
)

// @title A Eso Voy API
// @version 1.0
// @description This is the API for the A Eso Voy application.
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description "Type 'Bearer' followed by a space and then your token."
func main() {
	var port int
	flag.IntVar(&port, "port", 8080, "go backend server port")
	flag.Parse()

	app, err := app.NewApplication()
	if err != nil {
		panic(err)
	}
	defer app.DB.Close()

	r := routes.SetupRoutes(app)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	app.Logger.Info(fmt.Sprintf("we are running on port %d", port))

	err = server.ListenAndServe()
	if err != nil {
		app.Logger.Error(fmt.Sprintf("%v", err))
	}
}
