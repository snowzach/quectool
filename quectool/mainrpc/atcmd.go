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
			// Build response into a single right-sized buffer so we issue
			// one Write call and one allocation total.
			status := response.Status.String()
			size := len(response.Command) + 2 + len(status) + 2
			for _, line := range response.Response {
				size += len(line) + 2
			}
			buf := make([]byte, 0, size)
			buf = append(buf, response.Command...)
			buf = append(buf, '\r', '\n')
			for _, line := range response.Response {
				buf = append(buf, line...)
				buf = append(buf, '\r', '\n')
			}
			buf = append(buf, status...)
			buf = append(buf, '\r', '\n')
			_, _ = w.Write(buf)
			return
		}

		render.JSON(w, http.StatusOK, response)

	}

}
