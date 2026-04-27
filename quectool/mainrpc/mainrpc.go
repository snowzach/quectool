package mainrpc

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"

	"github.com/snowzach/golib/log"

	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/auth"
	"github.com/snowzach/quectool/quectool/modem"
	"github.com/snowzach/quectool/quectool/session"
)

// Server is the API web server
type Server struct {
	logger   *slog.Logger
	router   chi.Router
	atserver atserver.ATServer

	upgrader websocket.Upgrader

	terminalCommand string
	terminalArgs    []string

	modem    modem.Modem
	sessions *session.Manager
	verifier *auth.Verifier

	// credPath is where the password-change endpoint persists the new
	// bcrypt hash (htpasswd-style "user:hash" line). Empty means the
	// endpoint refuses (no place to write).
	credPath string
}

// Setup wires the API routes onto router.
//
// Auth model:
//   - Public: /api/auth/login (issues a token), /api/auth/whoami (reports
//     the current session, anonymous = empty user). Both safe without auth
//     so the SPA can show its login form before establishing a session.
//   - Session-protected: every other /api/* route, including /api/atcmd,
//     /api/probe/*, /api/sysinfo, /api/terminal, /api/modem/*, /api/dashboard,
//     /api/auth/logout, and /debug/pprof/*. Tokens are accepted as either a
//     session cookie or an Authorization: Bearer header.
//
// Static files (mounted by cmd/server.go at /) are public — the SPA shell
// must load before login. The data behind it is gated by the session-
// protected /api/* endpoints.
func Setup(router chi.Router, ats atserver.ATServer, mdm modem.Modem,
	sessions *session.Manager, verifier *auth.Verifier,
	terminalCommand string, terminalArgs []string,
	credPath string) error {

	s := &Server{
		logger:   log.Logger.With("context", "mainrpc"),
		router:   router,
		atserver: ats,
		modem:    mdm,
		sessions: sessions,
		verifier: verifier,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		terminalCommand: terminalCommand,
		terminalArgs:    terminalArgs,
		credPath:        credPath,
	}

	// Public auth routes.
	s.router.Post("/api/auth/login", s.AuthLogin())
	s.router.Get("/api/auth/whoami", s.AuthWhoami())

	// Everything else under /api requires a session.
	s.router.Group(func(r chi.Router) {
		r.Use(s.sessions.Middleware)

		r.Post("/api/auth/logout", s.AuthLogout())
		r.Post("/api/auth/passwd", s.AuthChangePassword())

		r.Get("/api/atcmd", s.ATCmd())
		r.Get("/api/probe/ping", s.ProbePing())
		r.Get("/api/probe/http", s.ProbeHTTP())
		r.Get("/api/sysinfo", s.SysInfo())
		r.Get("/api/terminal", s.Terminal())

		r.Get("/api/modem/info", s.ModemInfo())
		r.Get("/api/modem/sim", s.ModemSim())
		r.Get("/api/modem/cell", s.ModemCell())
		r.Get("/api/modem/signal", s.ModemSignal())
		r.Get("/api/modem/settings", s.ModemSettings())
		r.Put("/api/modem/settings", s.ModemSettingsUpdate())
		r.Post("/api/modem/cell-lock", s.ModemCellLock())
		r.Post("/api/modem/reboot", s.ModemReboot())
		r.Post("/api/modem/data-path", s.ModemDataPath())
		r.Post("/api/modem/scan", s.ModemScan())
		r.Post("/api/modem/cell-survey", s.ModemCellSurvey())
		r.Get("/api/modem/sms", s.ModemSMSList())
		r.Post("/api/modem/sms", s.ModemSMSSend())
		r.Delete("/api/modem/sms/{index}", s.ModemSMSDelete())
		r.Delete("/api/modem/sms", s.ModemSMSDeleteBulk())
		r.Post("/api/modem/sms/mark-all-read", s.ModemSMSMarkAllRead())

		r.Get("/api/dashboard", s.Dashboard())

		r.Mount("/debug", middleware.Profiler())
	})

	return nil
}
