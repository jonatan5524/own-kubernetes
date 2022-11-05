package main

import (
	"fmt"
	"log"

	"github.com/jonatan5524/own-kubernetes/pkg"
	"github.com/jonatan5524/own-kubernetes/pkg/agent/api"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func initRoutes(e *echo.Echo) {
	e.POST("/pods", api.CreatePod)
	e.GET("/pods/:id/log", api.LogPod)
	e.GET("/pods", api.GetAllPods)
	e.DELETE("/pods/:id", api.DeletePod)
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
	if err := pkg.ExecuteCommand("containerd"); err != nil {
		panic(err)
	}
	log.Printf("containerd running\n")
}

func main() {
	startContainerd()

	e := echo.New()

	initMiddlewares(e)
	initRoutes(e)

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", api.PORT)))
}
