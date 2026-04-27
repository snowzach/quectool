package mainrpc

import (
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/snowzach/golib/log"
)

// resizeMsg is sent by the frontend over the websocket as a TextMessage to
// signal terminal resize events. Keystroke data is sent as BinaryMessage.
type resizeMsg struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func (s *Server) Terminal() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("Error upgrading websocket connection: ", "error", err)
			return
		}
		defer conn.Close()

		cmd := exec.Command(s.terminalCommand, s.terminalArgs...)
		query := r.URL.Query()
		// Extract initial size from query parameters (LINES, COLUMNS) so the
		// shell starts with a sensible size before any resize event arrives.
		var initRows, initCols uint16 = 24, 80
		if rows := query.Get("LINES"); rows != "" {
			if v, _ := parseUint16(rows); v > 0 {
				initRows = v
			}
		}
		if cols := query.Get("COLUMNS"); cols != "" {
			if v, _ := parseUint16(cols); v > 0 {
				initCols = v
			}
		}
		// Also forward query as env vars (preserves existing behavior).
		if len(query) > 0 {
			cmd.Env = make([]string, 0, len(query))
			for key, value := range query {
				kv := make([]byte, 0, len(key)+1+len(value[0]))
				kv = append(kv, key...)
				kv = append(kv, '=')
				kv = append(kv, value[0]...)
				cmd.Env = append(cmd.Env, string(kv))
			}
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid:    true,
			Setctty:   true,
			Pdeathsig: syscall.SIGKILL,
		}

		ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: initRows, Cols: initCols})
		if err != nil {
			log.Errorf("pty start error: %v", err)
			return
		}
		defer func() { _ = ptmx.Close() }()

		// WS read loop: BinaryMessage = stdin bytes, TextMessage = JSON resize.
		// On any read error, kill the child so the writer goroutine and
		// cmd.Wait below unblock promptly.
		go func() {
			for {
				mt, message, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Errorf("WebSocket error: %v", err)
					}
					if cmd.Process != nil {
						_ = cmd.Process.Kill()
					}
					return
				}
				switch mt {
				case websocket.BinaryMessage, websocket.TextMessage:
					// Frontend may send resize as a JSON text message; detect
					// by leading '{' to avoid breaking older clients that
					// send keystrokes as TextMessage.
					if mt == websocket.TextMessage && len(message) > 0 && message[0] == '{' {
						var rs resizeMsg
						if err := json.Unmarshal(message, &rs); err == nil && rs.Cols > 0 && rs.Rows > 0 {
							_ = pty.Setsize(ptmx, &pty.Winsize{Rows: rs.Rows, Cols: rs.Cols})
							continue
						}
					}
					if _, err := ptmx.Write(message); err != nil {
						log.Errorf("Failed to write to pty: %v", err)
						if cmd.Process != nil {
							_ = cmd.Process.Kill()
						}
						return
					}
				}
			}
		}()

		// PTY → WS pump.
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := ptmx.Read(buf)
				if n > 0 {
					if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
						log.Errorf("Failed to write to WebSocket: %v", werr)
						return
					}
				}
				if err != nil {
					if err != io.EOF {
						log.Errorf("Failed to read from pty: %v", err)
					}
					return
				}
			}
		}()

		if err := cmd.Wait(); err != nil {
			log.Errorf("Command finished with error: %v", err)
		}

		// Best-effort close frame; conn.Close runs in defer.
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

		// Brief delay so the close frame can flush before the deferred
		// conn.Close. 50ms is plenty on local networks.
		time.Sleep(50 * time.Millisecond)
	}
}

func parseUint16(s string) (uint16, error) {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + uint64(c-'0')
		if n > 0xffff {
			return 0xffff, nil
		}
	}
	return uint16(n), nil
}
