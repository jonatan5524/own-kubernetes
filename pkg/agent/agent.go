package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/jonatan5524/own-kubernetes/pkg/agent/api"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func initRoutes(e *echo.Echo) {
	e.POST("/pod", api.CreatePod)
	e.GET("/pod/:id/log", api.LogPod)
	e.GET("/pod", api.GetAllPods)
	e.DELETE("/pod/:id", api.DeletePod)
}

func initMiddlewares(e *echo.Echo) {
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		c.Logger().Error(err)

		e.DefaultHTTPErrorHandler(err, c)
	}
}

func startContainerd() {
	cmd := exec.Command("containerd")
	cmd.Stdout = os.Stdout
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("containerd run on %d", cmd.Process.Pid)
}

func main() {
	startContainerd()

	e := echo.New()

	initMiddlewares(e)
	initRoutes(e)

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", api.PORT)))
}
