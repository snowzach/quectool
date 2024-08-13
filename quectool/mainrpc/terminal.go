package mainrpc

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/snowzach/golib/log"
)

func (s *Server) Terminal() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("Error upgrading websocket connection: ", "error", err)
			return
		}
		defer conn.Close()

		cmd := exec.CommandContext(ctx, s.terminalCommand, s.terminalArgs...)
		for key, value := range r.URL.Query() {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value[0]))
		}
		cmd.WaitDelay = 2 * time.Second
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Pdeathsig: syscall.SIGKILL,
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Errorf("stdin pipe error: %v", err)
			return
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Errorf("stdout pipe error: %v", err)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Errorf("stderr pipe error: %v", err)
			return
		}

		if err := cmd.Start(); err != nil {
			log.Error("Error running command: %v", err)
			return
		}

		// Goroutine to read from WebSocket and write to command stdin
		go func() {
			defer stdin.Close()
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Errorf("WebSocket error: %v", err)
					}
					break
				}
				_, err = stdin.Write(message)
				if err != nil {
					log.Errorf("Failed to write to stdin: %v", err)
					break
				}
			}
		}()

		// Goroutine to read from Bash's stdout and write to WebSocket
		go func() {
			defer stdout.Close()
			buf := make([]byte, 1024)
			for {
				n, err := stdout.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Errorf("Failed to read from stdout: %v", err)
					}
					break
				}
				err = conn.WriteMessage(websocket.TextMessage, buf[:n])
				if err != nil {
					log.Errorf("Failed to write to WebSocket: %v", err)
					break
				}
			}
		}()

		// Goroutine to read from Bash's stderr and write to WebSocket
		go func() {
			defer stderr.Close()
			buf := make([]byte, 1024)
			for {
				n, err := stderr.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Errorf("Failed to read from stderr: %v", err)
					}
					break
				}
				err = conn.WriteMessage(websocket.TextMessage, buf[:n])
				if err != nil {
					log.Errorf("Failed to write to WebSocket: %v", err)
					break
				}
			}
		}()

		// Wait for the command to finish
		if err := cmd.Wait(); err != nil {
			log.Errorf("Command finished with error: %v", err)
		}

		// Close WebSocket connection
		err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Errorf("Failed to close WebSocket connection: %v", err)
		}
	}
}
