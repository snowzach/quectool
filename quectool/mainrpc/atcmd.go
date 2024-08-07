package mainrpc

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/snowzach/golib/httpserver/render"
)

func (s *Server) ATCmd() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		query := r.URL.Query()
		cmd := query.Get("atcmd")
		if cmd == "" || !strings.HasPrefix(cmd, "AT") {
			render.ErrInvalidRequest(w, errors.New("invalid atcmd"))
			return
		}

		var (
			timeout time.Duration
			err     error
		)
		if timeoutString := query.Get("timeout"); timeoutString != "" {
			timeout, err = time.ParseDuration(timeoutString)
			if err != nil {
				render.ErrInvalidRequest(w, err)
				return
			}
		}

		response, err := s.atserver.SendCMD(ctx, cmd, timeout)
		if err != nil {
			render.ErrInvalidRequest(w, err)
			return
		}

		if query.Get("format") == "raw" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(response.Command + "\r\n"))
			for _, line := range response.Response {
				_, _ = w.Write([]byte(line + "\r\n"))
			}
			_, _ = w.Write([]byte(response.Status.String() + "\r\n"))
			return
		}

		render.JSON(w, http.StatusOK, response)

	}

}
