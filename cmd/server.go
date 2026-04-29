package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	cli "github.com/spf13/cobra"

	"github.com/snowzach/golib/conf"
	"github.com/snowzach/golib/httpserver"
	"github.com/snowzach/golib/httpserver/logger"
	"github.com/snowzach/golib/log"
	"github.com/snowzach/golib/signal"
	"github.com/snowzach/golib/version"
	"github.com/snowzach/quectool/embed"
	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/auth"
	"github.com/snowzach/quectool/quectool/cidrallow"
	"github.com/snowzach/quectool/quectool/credfile"
	"github.com/snowzach/quectool/quectool/tlsgen"
	"github.com/snowzach/quectool/quectool/iptables"
	"github.com/snowzach/quectool/quectool/mainrpc"
	"github.com/snowzach/quectool/quectool/modem"
	_ "github.com/snowzach/quectool/quectool/modem/generic" // register fallback
	_ "github.com/snowzach/quectool/quectool/modem/rm520"   // register impl
	"github.com/snowzach/quectool/quectool/session"
	"github.com/snowzach/quectool/quectool/sshserver"
)

func init() {
	rootCmd.AddCommand(apiCmd)
}

var (
	apiCmd = &cli.Command{
		Use:   "server",
		Short: "Start Server",
		Long:  `Start Server`,
		Run: func(cmd *cli.Command, args []string) { // Initialize the databse

			// Tune the runtime for a small-memory device. GOMEMLIMIT lets the
			// GC pace itself against an actual budget, and a low GOGC keeps
			// the heap close to the live set. Both can be overridden via
			// environment variables.
			tuneRuntimeForLowMemory()

			// Source-IP allow-list. The UI / SSH listen on every interface
			// (server.host = 0.0.0.0) for simplicity, but this matcher gates
			// every accepted connection — a request from outside the
			// allowed CIDRs gets a 403 (HTTP) or a TCP close (SSH) without
			// reaching auth, the router, or any business logic.
			allowMatcher, err := cidrallow.New(conf.C.Strings("server.allow_cidrs"))
			if err != nil {
				log.Fatalf("server.allow_cidrs: %v", err)
			}

			// Create the router and server config
			router, err := newRouter(allowMatcher)
			if err != nil {
				log.Fatalf("router config error: %v", err)
			}

			// Single credential source for HTTP login and SSH password auth.
			//
			// Precedence: server.auth.credentials_file (if it exists) overrides
			// server.auth.username/password from the YAML/env. The credentials
			// file is the only thing the UI password-change flow ever writes,
			// so the YAML stays purely declarative and an env-var override
			// can never silently shadow a UI-changed password. Reset path:
			// delete the file, restart.
			authUser := conf.C.String("server.auth.username")
			authHash := conf.C.String("server.auth.password")
			credPath := conf.C.String("server.auth.credentials_file")
			if credPath != "" {
				if u, h, err := credfile.Read(credPath); err == nil {
					log.Infof("auth: using credentials file %s (server.auth.password ignored)", credPath)
					authUser = u
					authHash = h
				} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, credfile.ErrEmpty) {
					log.Warnf("auth: ignoring credentials file %s: %v", credPath, err)
				}
			}
			verifier := auth.New(map[string]string{authUser: authHash})

			// Version endpoint (public).
			router.Get("/version", version.GetVersion())

			ipt, err := iptables.NewIPTables()
			if err != nil {
				log.Fatalf("could not setup iptables: %v", err)
			}

			// Set TTL
			if ttl := conf.C.Int("firewall.mangle.ttl"); ttl > 0 {
				if err := ipt.SetTTLValue(ttl); err != nil {
					log.Fatalf("could not setup ttl: %v", err)
				}
			}

			// Set up firewall
			if conf.C.Bool("firewall.filter.enabled") {
				if err := ipt.AllowTCPPorts(conf.C.Strings("firewall.filter.interfaces"), conf.C.Ints("firewall.filter.ports")); err != nil {
					log.Fatalf("could not setup tcp ports: %v", err)
				}
			}

			atserver, err := atserver.NewATServer(conf.C.String("modem.port"), conf.C.Duration("modem.timeout"))
			if err != nil {
				log.Fatalf("could not create AT server: %v", err)
			}

			detectCtx, detectCancel := context.WithTimeout(context.Background(), 10*time.Second)
			mdm, err := modem.Detect(detectCtx, atserver)
			detectCancel()
			if err != nil {
				port := conf.C.String("modem.port")
				log.Fatalf("could not detect modem on %s: %v\n"+
					"  - check the modem is powered and the AT port is correct (often /dev/ttyUSB2 or /dev/ttyUSB3)\n"+
					"  - check no other process is using the port (lsof %s, or fuser %s)\n"+
					"  - if the modem was left in prompt-input mode (e.g. half-finished CMGS), send ESC: printf '\\x1b' > %s",
					port, err, port, port, port)
			}
			log.Infof("modem detected: %s %s firmware=%s imei=%s",
				mdm.Info().Manufacturer, mdm.Info().Model, mdm.Info().Firmware, mdm.Info().IMEI)

			sessions := session.NewManager(24 * time.Hour)
			go func() {
				t := time.NewTicker(5 * time.Minute)
				defer t.Stop()
				for range t.C {
					sessions.Cleanup()
				}
			}()

			// MainRPC
			if err = mainrpc.Setup(router, atserver, mdm, sessions, verifier,
				conf.C.String("server.terminal.command"), conf.C.Strings("server.terminal.args"),
				credPath); err != nil {
				log.Fatalf("Could not setup mainrpc: %v", err)
			}

			mime.AddExtensionType(".css", "text/css")
			mime.AddExtensionType(".js", "application/javascript")

			var filesystem fs.FS
			if conf.C.Bool("server.embedded") {
				filesystem = embed.PublicHTMLFS()
			} else {
				filesystem = os.DirFS(conf.C.String("server.html_dir"))
			}
			// SPA static files. Real files (CSS/JS/index.html) are served as-is;
			// missing paths under non-/api fall back to index.html so client-side
			// routes survive a hard refresh.
			htmlFilesServer := http.FileServer(http.FS(filesystem))
			router.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/" {
					if f, err := filesystem.Open(strings.TrimPrefix(r.URL.Path, "/")); err == nil {
						_ = f.Close()
						w.Header().Set("Vary", "Accept-Encoding")
						w.Header().Set("Cache-Control", "no-cache")
						htmlFilesServer.ServeHTTP(w, r)
						return
					}
					// Real 404 for /api so missing endpoints don't masquerade as the SPA.
					if strings.HasPrefix(r.URL.Path, "/api/") {
						http.NotFound(w, r)
						return
					}
				}
				// Fall through: serve index.html.
				f, err := filesystem.Open("index.html")
				if err != nil {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				defer f.Close()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache")
				_, _ = io.Copy(w, f)
			}))

			// Auto-generate a self-signed cert on first boot if TLS is on
			// and the configured paths don't exist yet. Has to happen before
			// newServer() because httpserver loads cert/key eagerly when
			// building its TLSConfig.
			if conf.C.Bool("server.tls") {
				certPath := conf.C.String("server.certfile")
				keyPath := conf.C.String("server.keyfile")
				res, generated, err := tlsgen.EnsureExists(tlsgen.Config{
					CertPath: certPath,
					KeyPath:  keyPath,
				})
				if err != nil {
					log.Fatalf("could not ensure TLS cert/key: %v", err)
				}
				if generated {
					log.Infof("generated self-signed TLS cert at %s (valid until %s, hosts: %v)",
						certPath, res.NotAfter.Format("2006-01-02"), res.Hosts)
				}
			}

			// Create a server
			s, err := newServer(router)
			if err != nil {
				log.Fatalf("could not create server error: %v", err)
			}

			// Bound per-connection memory. We deliberately do NOT set
			// ReadTimeout / WriteTimeout because the terminal endpoint
			// upgrades to a long-lived websocket. ReadHeaderTimeout caps
			// slow-loris-style header reads; IdleTimeout reaps idle keep-alive
			// connections (does not apply to hijacked/upgraded conns);
			// MaxHeaderBytes caps the per-conn header buffer.
			s.ReadHeaderTimeout = 10 * time.Second
			s.IdleTimeout = 60 * time.Second
			s.MaxHeaderBytes = 16 << 10 // 16 KiB

			// Start the listener and service connections.
			go func() {
				// Override the default listener to ensure we only listen on IPv4
				listener, err := net.Listen("tcp4", s.Addr)
				if err != nil {
					log.Fatalf("could not listen on %s: %v", s.Addr, err)
				}

				// Enable TLS?
				if conf.C.Bool("server.tls") {
					// Wrap the listener in a TLS Listener
					listener = tls.NewListener(listener, s.TLSConfig)
				}

				if err := s.Serve(listener); err != nil {
					log.Errorf("Server error: %v", err)
					signal.Stop.Stop()
				}
			}()
			log.Infof("API listening on %s", s.Addr)

			// Embedded SSH server.
			var sshSrv *sshserver.Server
			if conf.C.Bool("server.ssh.enabled") {
				sshSrv, err = sshserver.New(&sshserver.Config{
					Addr:           conf.C.String("server.ssh.address"),
					HostKeyFile:    conf.C.String("server.ssh.host_key_file"),
					AuthorizedKeys: conf.C.String("server.ssh.authorized_keys"),
					Shell:          conf.C.String("server.ssh.shell"),
					ShellArgs:      conf.C.Strings("server.ssh.shell_args"),
					IdleTimeout:    conf.C.Duration("server.ssh.idle_timeout"),
					Verifier:       verifier,
					Allow:          allowMatcher,
				})
				if err != nil {
					log.Fatalf("could not create ssh server: %v", err)
				}
				go func() {
					if err := sshSrv.ListenAndServe(); err != nil &&
						err.Error() != "ssh: Server closed" {
						log.Errorf("ssh server error: %v", err)
					}
				}()
			}

			// Register signal handler and wait
			signal.Stop.OnSignal(signal.DefaultStopSignals...)
			<-signal.Stop.Chan() // Wait until Stop

			if sshSrv != nil {
				_ = sshSrv.Close()
			}
			_ = atserver.Close()

			signal.Stop.Wait() // Wait until everyone cleans up
		},
	}
)

func newRouter(allow *cidrallow.Matcher) (chi.Router, error) {

	router := chi.NewRouter()
	router.Use(
		middleware.Recoverer, // Recover from panics
		middleware.RequestID, // Inject request-id
		allow.Middleware,     // Source-IP allow-list (must come before logging/cors)
	)

	// Request logger
	if conf.C.Bool("server.log.enabled") {
		var loggerConfig logger.Config
		if err := conf.C.Unmarshal(&loggerConfig, conf.UnmarshalConf{Path: "server.log"}); err != nil {
			return nil, fmt.Errorf("could not parser server.log config: %w", err)
		}
		router.Use(logger.LoggerStandardMiddleware(log.Logger.With("context", "server"), loggerConfig))
	}

	// CORS handler
	if conf.C.Bool("server.cors.enabled") {
		var corsOptions cors.Options
		if err := conf.C.Unmarshal(&corsOptions, conf.UnmarshalConf{
			Path: "server.cors",
			DecoderConfig: conf.DefaultDecoderConfig(
				conf.WithMatchName(conf.MatchSnakeCaseConfig),
			),
		}); err != nil {
			return nil, fmt.Errorf("could not parser server.cors config: %w", err)
		}
		router.Use(cors.New(corsOptions).Handler)
	}

	return router, nil

}

func newServer(handler http.Handler) (*httpserver.Server, error) {

	// Parse the config
	var serverConfig = &httpserver.Config{Handler: handler}
	if err := conf.C.Unmarshal(serverConfig, conf.UnmarshalConf{Path: "server"}); err != nil {
		return nil, fmt.Errorf("could not parse server config: %w", err)
	}

	// Create the server
	s, err := httpserver.New(httpserver.WithConfig(serverConfig))
	if err != nil {
		return nil, fmt.Errorf("could not create server: %w", err)
	}

	return s, nil

}

// tuneRuntimeForLowMemory configures the Go runtime for small-memory devices
// like the Quectel modem. It only sets values that aren't already controlled
// by the user's environment (GOMEMLIMIT / GOGC), and runs a single startup
// scavenge to release init-time allocations back to the OS.
func tuneRuntimeForLowMemory() {
	// 32 MiB soft cap. The runtime will pace GC against this budget.
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(32 << 20)
	}
	// GC at 50% heap growth — middle ground between default (100%) and
	// aggressive (20%). With a 32 MiB cap, the limit itself does most of
	// the pacing work; this just keeps the live heap closer to the working set.
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(50)
	}
	// One-shot scavenge to release init-time pages.
	debug.FreeOSMemory()
}
