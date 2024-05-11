package cmd

import "net/http"

// defaults loads the default config for the app
func defaults() map[string]interface{} {
	return map[string]interface{}{
		// Logger Defaults
		"logger.level":    "info",
		"logger.encoding": "console",
		"logger.color":    true,
		"logger.output":   "stderr",

		"pidfile": "",

		// Server Configuration
		"server.host":     "0.0.0.0",
		"server.port":     "8080",
		"server.tls":      false,
		"server.devcert":  false,
		"server.certfile": "server.crt",
		"server.keyfile":  "server.key",
		// Server Log
		"server.log.enabled":       true,
		"server.log.level":         "info",
		"server.log.request_body":  false,
		"server.log.response_body": false,
		"server.log.ignore_paths":  []string{"/version"},
		// Server CORS
		"server.cors.enabled":           true,
		"server.cors.allowed_origins":   []string{"*"},
		"server.cors.allowed_methods":   []string{http.MethodHead, http.MethodOptions, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
		"server.cors.allowed_headers":   []string{"*"},
		"server.cors.allow_credentials": false,
		"server.cors.max_age":           300,
		// Embedded Server or local filesystem for html
		"server.embedded": true,
		"server.html_dir": "embed/public_html",
		// Server Auth
		"server.auth.realm":    "",
		"server.auth.username": "admin",
		"server.auth.password": "password",
		"server.auth.htpasswd": "",

		"modem.port":    "/dev/smd11",
		"modem.timeout": "5s",
	}
}
