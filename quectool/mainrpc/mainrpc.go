package mainrpc

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/snowzach/golib/log"

	"github.com/snowzach/quectool/quectool/atserver"
)

// Server is the API web server
type Server struct {
	logger   *slog.Logger
	router   chi.Router
	atserver atserver.ATServer

	upgrader websocket.Upgrader

	terminalCommand string
	terminalArgs    []string
}

// Setup will setup the API listener
func Setup(router chi.Router, atserver atserver.ATServer, terminalCommand string, terminalArgs []string) error {

	s := &Server{
		logger:   log.Logger.With("context", "mainrpc"),
		router:   router,
		atserver: atserver,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		terminalCommand: terminalCommand,
		terminalArgs:    terminalArgs,
	}

	// Base Functions
	s.router.Route("/api", func(r chi.Router) {
		r.Get("/atcmd", s.ATCmd())
		r.Get("/probe/ping", s.ProbePing())
		r.Get("/probe/http", s.ProbeHTTP())
		r.Get("/sysinfo", s.SysInfo())
		r.Get("/terminal", s.Terminal())
	})

	return nil

}
