package mainrpc

import (
	"errors"
	"net/http"
	"time"

	"github.com/snowzach/golib/httpserver/render"
	"github.com/snowzach/quectool/quectool/prober"
)

func (s *Server) ProbePing() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		query := r.URL.Query()
		target := query.Get("target")
		if target == "" {
			render.ErrInvalidRequest(w, errors.New("invalid target"))
			return
		}

		result, err := prober.ProbePing(ctx, target, 3*time.Second)
		if err != nil {
			render.ErrInvalidRequest(w, err)
		}

		render.JSON(w, http.StatusOK, result)

	}

}

func (s *Server) ProbeHTTP() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		query := r.URL.Query()
		target := query.Get("target")
		if target == "" {
			render.ErrInvalidRequest(w, errors.New("invalid target"))
			return
		}

		result, err := prober.ProbeHTTP(ctx, target, 3*time.Second)
		if err != nil {
			render.ErrInvalidRequest(w, err)
		}

		render.JSON(w, http.StatusOK, result)

	}
}
