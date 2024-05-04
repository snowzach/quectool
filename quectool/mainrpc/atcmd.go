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
		cmd := query.Get("cmd")
		if cmd == "" || !strings.HasPrefix(cmd, "AT") {
			render.ErrInvalidRequest(w, errors.New("invalid cmd"))
			return
		}

		result, err := s.atserver.SendCMD(ctx, cmd)
		if err != nil {
			render.ErrInvalidRequest(w, err)
			return
		}

		render.JSON(w, http.StatusOK, result)

	}

}
