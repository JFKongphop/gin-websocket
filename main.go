package main

import (
	"github.com/gin-gonic/gin"
	"gin-websocket/config-websocket"
	
)

func main() {
	hub := configwebsocket.H
	go hub.Run()



	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)

	router.Use(gin.Logger())

	router.GET("/ws", func(c *gin.Context) {
		configwebsocket.ServeWs(hub, c.Writer, c.Request)
	})
	router.Run(":8000")
}