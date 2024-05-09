package mainrpc

import (
	"errors"
	"net/http"
	"strings"

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

		response, err := s.atserver.SendCMD(ctx, cmd)
		if err != nil {
			render.ErrInvalidRequest(w, err)
			return
		}

		if query.Get("format") == "raw" {
			for _, line := range response.Response {
				w.Write([]byte(line))
				w.Write([]byte("\r\n"))
			}
			w.Write([]byte(response.Status.String()))
			w.Write([]byte("\r\n"))
			return
		}

		render.JSON(w, http.StatusOK, response)

	}

}
