package cmd

import (
	"crypto/subtle"
	"crypto/tls"
	"fmt"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	cli "github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"github.com/snowzach/golib/conf"
	"github.com/snowzach/golib/httpserver"
	"github.com/snowzach/golib/httpserver/logger"
	"github.com/snowzach/golib/log"
	"github.com/snowzach/golib/signal"
	"github.com/snowzach/golib/version"
	"github.com/snowzach/quectool/embed"
	"github.com/snowzach/quectool/quectool/atserver"
	"github.com/snowzach/quectool/quectool/iptables"
	"github.com/snowzach/quectool/quectool/mainrpc"
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

			// Will run a Garbage Collection and return as much memory to the system as possible
			// every 5 seconds if the memory in the system is below 1GB.
			keepMemoryUsageLowIfNeeded()

			// Create the router and server config
			router, err := newRouter()
			if err != nil {
				log.Fatalf("router config error: %v", err)
			}

			// Simple creds
			realm := conf.C.String("server.auth.realm")
			creds := map[string]string{
				conf.C.String("server.auth.username"): conf.C.String("server.auth.password"),
			}
			router.Use(BasicAuth(realm, creds))

			// Version endpoint
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

			// MainRPC
			if err = mainrpc.Setup(router, atserver, conf.C.String("server.terminal.command"), conf.C.Strings("server.terminal.args")); err != nil {
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
			htmlFilesServer := http.FileServer(http.FS(filesystem))
			router.Mount("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Vary", "Accept-Encoding")
				w.Header().Set("Cache-Control", "no-cache")
				htmlFilesServer.ServeHTTP(w, r)
			}))

			if conf.C.Bool("server.ttyd.enabled") {
				path := conf.C.String("server.ttyd.path")
				target, _ := url.Parse(conf.C.String("server.ttyd.address"))
				router.Mount(path, http.StripPrefix(path, &httputil.ReverseProxy{
					Director: func(r *http.Request) {
						r.URL = target
					},
				}))
			}

			if conf.C.Bool("server.pprof") {
				router.Mount("/debug", middleware.Profiler())
			}

			// Create a server
			s, err := newServer(router)
			if err != nil {
				log.Fatalf("could not create server error: %v", err)
			}

			// Start the listener and service connections.
			go func() {
				// Override the default listener to ensure we only listen on IPv4
				listener, err := net.Listen("tcp4", s.Addr)
				if err != nil {
					log.Fatalf("could not listen on %s: %w", s.Addr, err)
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

			// Register signal handler and wait
			signal.Stop.OnSignal(signal.DefaultStopSignals...)
			<-signal.Stop.Chan() // Wait until Stop

			_ = atserver.Close()

			signal.Stop.Wait() // Wait until everyone cleans up
		},
	}
)

func newRouter() (chi.Router, error) {

	router := chi.NewRouter()
	router.Use(
		middleware.Recoverer, // Recover from panics
		middleware.RequestID, // Inject request-id
	)

	// Request logger
	if conf.C.Bool("server.log.enabled") {
		var loggerConfig logger.Config
		if err := conf.C.Unmarshal(&loggerConfig, conf.UnmarshalConf{Path: "server.log"}); err != nil {
			return nil, fmt.Errorf("could not parser server.log config: %w", err)
		}
		switch conf.C.String("logger.encoding") {
		default:
			router.Use(logger.LoggerStandardMiddleware(log.Logger.With("context", "server"), loggerConfig))
		}
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

// BasicAuth implements a simple middleware handler for adding basic http auth to a route.
func BasicAuth(realm string, creds map[string]string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok {
				basicAuthFailed(w, realm)
				return
			}

			credPass, credUserOk := creds[user]

			if credUserOk {
				if strings.HasPrefix(credPass, "$2") {
					// bcrypt hash
					if err := bcrypt.CompareHashAndPassword([]byte(credPass), []byte(pass)); err == nil {
						next.ServeHTTP(w, r)
						return
					}
				} else if subtle.ConstantTimeCompare([]byte(pass), []byte(credPass)) == 1 {
					next.ServeHTTP(w, r)
					return
				}
			}
			basicAuthFailed(w, realm)
		})
	}
}

func basicAuthFailed(w http.ResponseWriter, realm string) {
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	w.WriteHeader(http.StatusUnauthorized)
}

func keepMemoryUsageLowIfNeeded() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	if memStats.Sys < 1<<30 { // 1 GB
		log.Info("Sceduled garbage collection every 5 seconds")
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				debug.FreeOSMemory()
			}
		}()
	}

}
