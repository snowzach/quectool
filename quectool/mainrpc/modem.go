package mainrpc

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/snowzach/quectool/quectool/modem"
)

func (s *Server) ModemInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.modem.Info())
	}
}

func (s *Server) ModemSim() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := s.modem.GetSim(r.Context())
		writeModem(w, v, err)
	}
}

func (s *Server) ModemCell() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := s.modem.GetCell(r.Context())
		writeModem(w, v, err)
	}
}

func (s *Server) ModemSignal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := s.modem.GetSignal(r.Context())
		writeModem(w, v, err)
	}
}

func (s *Server) ModemSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := s.modem.GetSettings(r.Context())
		writeModem(w, v, err)
	}
}

func (s *Server) ModemSettingsUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u modem.SettingsUpdate
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			writeErr(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid json")
			return
		}
		v, err := s.modem.SetSettings(r.Context(), &u)
		writeModem(w, v, err)
	}
}

type cellLockReq struct {
	Tech string `json:"tech"`            // "4g" or "5g"
	Freq int    `json:"freq,omitempty"`  // EARFCN/ARFCN; if 0, lock to current cell
	PCI  int    `json:"pci,omitempty"`
	SCS  int    `json:"scs,omitempty"`   // NR5G QENG/QSCAN scs index (0=15kHz, 1=30kHz, ...)
	Band int    `json:"band,omitempty"`  // numeric band (e.g. 71 for n71); required for 5G
}

func (s *Server) ModemCellLock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b cellLockReq
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeErr(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid json")
			return
		}
		var err error
		if b.Freq > 0 {
			err = s.modem.LockCell(r.Context(), b.Tech, b.Freq, b.PCI, b.SCS, b.Band)
		} else {
			err = s.modem.LockCurrentCell(r.Context(), b.Tech)
		}
		if err != nil {
			writeModem(w, nil, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

type dataPathReq struct {
	Path string `json:"path"` // "USB" or "PCIe"
}

func (s *Server) ModemDataPath() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b dataPathReq
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeErr(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid json")
			return
		}
		if err := s.modem.SetDataPath(r.Context(), b.Path); err != nil {
			writeModem(w, nil, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) ModemReboot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.modem.Reboot(r.Context()); err != nil {
			writeModem(w, nil, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) ModemScan() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := s.modem.Scan(r.Context())
		writeModem(w, v, err)
	}
}

func (s *Server) ModemCellSurvey() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := s.modem.CellSurvey(r.Context())
		writeModem(w, v, err)
	}
}

func (s *Server) ModemSMSList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := s.modem.ListSMS(r.Context())
		writeModem(w, v, err)
	}
}

type sendSMSReq struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

func (s *Server) ModemSMSSend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b sendSMSReq
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeErr(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid json")
			return
		}
		if err := s.modem.SendSMS(r.Context(), b.To, b.Body); err != nil {
			writeModem(w, nil, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) ModemSMSDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idx, err := strconv.Atoi(chi.URLParam(r, "index"))
		if err != nil {
			writeErr(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "invalid index")
			return
		}
		if err := s.modem.DeleteSMS(r.Context(), idx); err != nil {
			writeModem(w, nil, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) ModemSMSDeleteBulk() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope := r.URL.Query().Get("scope")
		if scope != "read" && scope != "all" {
			writeErr(w, http.StatusBadRequest, "ERR_BAD_REQUEST", "scope must be read or all")
			return
		}
		if err := s.modem.DeleteSMSBulk(r.Context(), scope); err != nil {
			writeModem(w, nil, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) ModemSMSMarkAllRead() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.modem.MarkAllRead(r.Context()); err != nil {
			writeModem(w, nil, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeModem(w http.ResponseWriter, v any, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, v)
		return
	}
	switch {
	case errors.Is(err, modem.ErrUnsupported):
		writeErr(w, http.StatusMethodNotAllowed, "ERR_UNSUPPORTED", err.Error())
	case errors.Is(err, modem.ErrFieldMissing):
		writeErr(w, http.StatusBadGateway, "ERR_FIELD_MISSING", err.Error())
	default:
		writeErr(w, http.StatusBadGateway, "ERR_MODEM", err.Error())
	}
}
