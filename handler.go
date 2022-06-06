package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Serv struct {
	httpServer *http.Server
}

func (s *Serv) Run(port string) error {
	handler := gin.New()
	handler.GET("/", Start)
	s.httpServer = &http.Server{
		Addr:    port,
		Handler: handler,
	}
	log.Printf("listen port %s", port)
	return s.httpServer.ListenAndServe()
}
