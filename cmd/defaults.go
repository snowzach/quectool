package cmd

import "net/http"

// defaults loads the default config for the app
func defaults() map[string]any {
	return map[string]any{
		// Logger Defaults
		"logger.level":    "info",
		"logger.encoding": "console",
		"logger.color":    true,
		"logger.output":   "stderr",

		"pidfile": "",

		// Server Configuration. HTTPS by default — the binary auto-creates
		// a self-signed cert at server.certfile / server.keyfile on first
		// boot if they don't exist, with SAN entries for every UP-interface
		// IP. To regenerate after a network reconfiguration: delete the
		// files and restart, or run `quectool gencert --force`.
		"server.host":     "0.0.0.0",
		"server.port":     "443",
		"server.tls":      true,
		"server.devcert":  false,
		"server.certfile": "/usrdata/quectool/server.crt",
		"server.keyfile":  "/usrdata/quectool/server.key",
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

		// Server Auth (used for /api/auth/login and SSH password auth).
		// Username + password here are the BOOTSTRAP credentials. Once an
		// operator changes the password from the web UI, the new bcrypt
		// hash is written to server.auth.credentials_file and that file
		// wins on subsequent restarts. To reset: delete the file.
		"server.auth.username":         "admin",
		"server.auth.password":         "$2a$10$7mmv.UY3blHzjamcvvQ1QO8zFv6/46VPHmUyyogoJv6g7E73J5Sym", // password = quectool
		"server.auth.credentials_file": "/usrdata/quectool/credentials",

		"server.terminal.command": "/bin/bash",
		"server.terminal.args":    []string{"-i", "-l"},

		// Embedded SSH server. Reuses server.auth credentials for password
		// auth. Host key is generated on first run and persisted.
		"server.ssh.enabled":         true,
		"server.ssh.address":         "0.0.0.0:2222",
		"server.ssh.host_key_file":   "host_key",
		"server.ssh.authorized_keys": "",
		"server.ssh.shell":           "/bin/bash",
		"server.ssh.shell_args":      []string{"-i", "-l"},
		"server.ssh.idle_timeout":    "30m",

		"firewall.filter.enabled":    false,
		"firewall.filter.interfaces": []string{"bridge", "eth0", "tailscale0"},
		"firewall.filter.ports":      []int{22, 80, 443, 8080},
		"firewall.mangle.ttl":        0,

		"modem.port":    "/dev/smd11",
		"modem.timeout": "5s",
	}
}
