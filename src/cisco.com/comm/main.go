package main

import (
	"cisco.com/comm/api"
	"cisco.com/comm/socket"
	"cisco.com/comm/log"
	"flag"
	"strings"
)

var Options struct {
	APIPort int
	Server  string
	Port    int
	Mode    string
	Output  string
}

func init() {
	// Command-line options
	what := flag.String(
		"mode",
		"server",
		"Mode of operation ('client', 'server')")
	output := flag.String(
		"handler",
		"api",
		"Connection handling mode for socket server. You will never need to override this unless you know what you're doing")
	port := flag.Int(
		"p",
		0,
		"Port to bind to (in server mode) or connect to (in client mode)")
	apiport := flag.Int(
		"apiport",
		0,
		"Port that the API should listen on")
	server := flag.String(
		"s",
		"localhost",
		"Server to connect to (only valid in client mode)",
	)

	flag.Parse()
	Options.Mode = *what
	Options.Port = *port
	Options.Output = *output
	Options.APIPort = *apiport
	Options.Server = *server
}

func main() {
	log.I("Booting. Using configuration %v", Options)

	switch strings.ToLower(Options.Mode) {
	case "server":
		runServer()
	case "client":
		runClient()
	default:
		log.F("Unknown mode", Options.Mode)
	}
}

func getHandler() socket.ConnectionHandler {
	switch Options.Output {
	case "echo":
		log.D("Using echo handler")
		return &socket.EchoHandler{}
	case "api":
		log.D("Using API handler")
		return socket.NewChannelHandler()
	default:
		log.F("Unknown handler")
		return nil
	}
}

func runServer() {
	errc := make(chan error)
	handler := getHandler()
	socketServer := socket.NewServer(Options.Port, handler)
	apiServer := api.APIServer{Port: Options.APIPort, SocketServer: socketServer}

	// Start servers and wait for termination
	go func() {
		errc <- apiServer.Listen()
	}()

	go func() {
		errc <- socketServer.Listen()
	}()

	err := <-errc
	if err != nil {
		log.F("Failed to start server daemons %v", err)
	}
}

func runClient() {
	log.I("Starting in CLIENT mode. Connecting to %s:%d", "localhost", Options.Port)
	errc := make(chan error)
	handler := getHandler()

	if Options.Server == "" {
		log.F("No server given")
	}

	cSocketServer, err := socket.NewClient(Options.Server, Options.Port, handler)

	if err != nil {
		log.F("Failed to start client %v", err)
	}

	apiserver := api.APIServer{Port: Options.APIPort, SocketServer: cSocketServer}

	// Start servers and wait for termination
	go func() {
		errc <- cSocketServer.Connect()
	}()

	go func() {
		errc <- apiserver.Listen()
	}()

	err = <-errc

	if err != nil {
		log.F("Failed to start daemons %v", err)
	}
}
