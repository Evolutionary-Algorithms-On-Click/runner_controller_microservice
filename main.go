package main

import (
	"evolve/config"
	"evolve/controller"
	"evolve/routes"
	"evolve/util"
	"fmt"
	"net/http"
)

func main() {
	var logger = util.NewLogger()

	// Register routes.
	http.HandleFunc(routes.TEST, controller.Test)
	http.HandleFunc(routes.EA, controller.CreateEA)
	http.HandleFunc(routes.GP, controller.CreateGP)
	http.HandleFunc(routes.ML, controller.CreateML)

	logger.Info(fmt.Sprintf("Test http server on http://localhost%v/api/test", config.PORT))

	if err := http.ListenAndServe(config.PORT, nil); err != nil {
		logger.Error(fmt.Sprintf("Failed to start server: %v", err))
		return
	}
}
