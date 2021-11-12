package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Response struct {
	Message string `json:"message" xml:"message"`
}

func main() {
	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}",` +
			`"host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}",` +
			`"status":${status},"error":"${error}","latency":${latency},"latency_human":"${latency_human}"` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
	}))

	e.GET("/", func(c echo.Context) error {
		u := &Response{
			Message: "Hello World",
		}
		return c.JSON(http.StatusOK, u)
	})

	e.GET("/error", func(c echo.Context) error {
		u := &Response{
			Message: "Error",
		}
		return c.JSON(http.StatusInternalServerError, u)
	})

	e.Logger.Fatal(e.Start(":1323"))
}
