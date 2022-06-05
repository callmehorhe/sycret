package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Server struct {
	httpServer *http.Server
}

func (s *Server) Run(port string) error {
	handler := gin.New()
	handler.GET("/", )
	s.httpServer = &http.Server{
		Addr: port,
		Handler: handler,
	}
	return s.httpServer.ListenAndServe()
}