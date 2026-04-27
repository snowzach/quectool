package mainrpc

import (
	"errors"
	"net/http"

	"github.com/snowzach/quectool/quectool/modem"
	"github.com/snowzach/quectool/quectool/sysinfo"
)

type dashboard struct {
	Info    modem.Info        `json:"info"`
	Sim     *modem.SimInfo    `json:"sim,omitempty"`
	Cell    *modem.CellInfo   `json:"cell,omitempty"`
	Signal  *modem.Signal     `json:"signal,omitempty"`
	Sysinfo *sysinfo.SysInfo  `json:"sysinfo,omitempty"`
	Errors  map[string]string `json:"errors,omitempty"`
}

func (s *Server) Dashboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		out := &dashboard{Info: s.modem.Info(), Errors: map[string]string{}}

		// ErrUnsupported just means the modem can't speak that surface (e.g.
		// generic fallback) — not an error worth showing the user. Render
		// only real failures.
		record := func(key string, err error) {
			if err == nil || errors.Is(err, modem.ErrUnsupported) {
				return
			}
			out.Errors[key] = err.Error()
		}

		v, err := s.modem.GetSim(ctx)
		out.Sim = v
		record("sim", err)
		c, err := s.modem.GetCell(ctx)
		out.Cell = c
		record("cell", err)
		sig, err := s.modem.GetSignal(ctx)
		out.Signal = sig
		record("signal", err)
		si, err := sysinfo.Get(ctx)
		out.Sysinfo = si
		record("sysinfo", err)
		if len(out.Errors) == 0 {
			out.Errors = nil
		}
		writeJSON(w, http.StatusOK, out)
	}
}
