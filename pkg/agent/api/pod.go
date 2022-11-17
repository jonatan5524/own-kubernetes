package api

import (
	"net/http"

	"github.com/jonatan5524/own-kubernetes/pkg/pod"
	"github.com/labstack/echo/v4"
)

type podDTO struct {
	ImageRegistry string `json:"image registry"`
	Name          string `json:"name"`
}

func CreatePod(c echo.Context) error {
	podDto := new(podDTO)
	if err := c.Bind(podDto); err != nil {
		return err
	}

	runningPod, err := pod.NewPodAndRun(podDto.ImageRegistry, podDto.Name)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, podDTO{
		ImageRegistry: podDto.ImageRegistry,
		Name:          runningPod.Pod.Id,
	})
}

func LogPod(c echo.Context) error {
	logs, err := pod.LogPod(c.Param("id"))
	if err != nil {
		return err
	}

	return c.String(http.StatusOK, logs)
}

func GetAllPods(c echo.Context) error {
	pods, err := pod.ListRunningPods()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, pods)
}

func DeletePod(c echo.Context) error {
	if _, err := pod.KillPod(c.Param("id")); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)

}
