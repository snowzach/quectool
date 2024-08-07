package mainrpc

import (
	"net/http"

	"github.com/snowzach/golib/httpserver/render"
	"github.com/snowzach/quectool/quectool/sysinfo"
)

func (s *Server) SysInfo() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		result, err := sysinfo.Get(ctx)
		if err != nil {
			render.ErrInvalidRequest(w, err)
			return
		}

		render.JSON(w, http.StatusOK, result)

	}
}
