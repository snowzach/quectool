package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/posflag"
	"github.com/snowzach/golib/conf"
	"github.com/snowzach/golib/log"
	cli "github.com/spf13/cobra"

	"github.com/snowzach/quectool/quectool/atserver"
)

func init() {
	rootCmd.AddCommand(atCmd)
	atCmd.PersistentFlags().StringP("modem.port", "p", "", "Modem Port")
}

var (
	atCmd = &cli.Command{
		Use:   "atcmd",
		Short: "Send AT Command",
		Long:  `Start Server`,
		RunE: func(cmd *cli.Command, args []string) error { // Initialize the databse

			if len(args) == 0 {
				return errors.New("no command provided")
			}

			if err := conf.C.Load(posflag.Provider(cmd.Flags(), ".", conf.C.Koanf), nil); err != nil {
				log.Fatalf("could not load configuration: %v", err)
			}

			ats, err := atserver.NewATServer(conf.C.String("modem.port"), conf.C.Duration("modem.timeout"))
			if err != nil {
				log.Fatalf("could not create AT server: %v", err)
			}
			defer ats.Close()

			var response *atserver.ATResponse
			for retries := 0; retries < 3; retries++ {
				response, err = ats.SendCMD(context.Background(), args[0], 10*time.Second)
				if err != nil {
					if strings.Contains(err.Error(), "timeout") {
						continue
					}
					log.Fatalf("could not send command: %v", err)
				}
				break
			}

			for _, line := range response.Response {
				fmt.Println(line)
			}
			fmt.Println(response.Status.String())

			return nil
		},
	}
)
