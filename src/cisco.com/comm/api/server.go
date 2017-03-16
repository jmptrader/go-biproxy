package api

import (
	"cisco.com/comm/log"
	"fmt"
	"net/http"
)

type APIServer struct {
	Port         int
	SocketServer SocketServer
}

func (a *APIServer) Listen() error {
	root := Controller{Server: a.SocketServer}
	log.I("Starting. Bind to TCP %d", a.Port)
	http.HandleFunc("/connections", root.Connections)
	return http.ListenAndServe(fmt.Sprintf(":%d", a.Port), nil)
}
