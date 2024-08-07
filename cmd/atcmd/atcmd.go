package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/snowzach/golib/httpserver/render"
	"github.com/snowzach/golib/log"
	"github.com/snowzach/quectool/quectool/atserver"
)

func main() {

	if err := log.InitLogger(&log.LoggerConfig{
		Level:    "info",
		Encoding: "text",
		Color:    true,
		Output:   "stdout",
	}); err != nil {
		fmt.Printf("could not configure logger: %v", err)
		os.Exit(1)
	}

	if len(os.Args) != 3 {
		log.Fatal("Usage: atcmd <modem port> <server listen>")
	}

	ats, err := atserver.NewATServer(os.Args[1], 5*time.Second)
	if err != nil {
		log.Fatalf("could not create AT server: %v", err)
	}
	defer ats.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/atcmd", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query()
		cmd := query.Get("atcmd")
		if cmd == "" || !strings.HasPrefix(cmd, "AT") {
			render.ErrInvalidRequest(w, errors.New("invalid atcmd"))
			return
		}

		response, err := ats.SendCMD(ctx, cmd, 10*time.Second)
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

	})

	log.Infof("Starting server on %s", os.Args[2])

	http.ListenAndServe(os.Args[2], mux)

}
