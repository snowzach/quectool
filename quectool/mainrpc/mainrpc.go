package mainrpc

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/snowzach/golib/log"

	"github.com/snowzach/quectool/quectool/atserver"
)

// Server is the API web server
type Server struct {
	logger   *slog.Logger
	router   chi.Router
	atserver atserver.ATServer
}

// Setup will setup the API listener
func Setup(router chi.Router, atserver atserver.ATServer, realm string, creds map[string]string) error {

	s := &Server{
		logger:   log.Logger.With("context", "mainrpc"),
		router:   router,
		atserver: atserver,
	}

	// Base Functions
	s.router.Route("/api", func(r chi.Router) {
		r.Get("/atcmd", s.ATCmd())
		r.Get("/probe/ping", s.ProbePing())
		r.Get("/probe/http", s.ProbeHTTP())
		r.Get("/sysinfo", s.SysInfo())
	})

	return nil

}
