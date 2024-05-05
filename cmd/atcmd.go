package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/snowzach/golib/conf"
	"github.com/snowzach/golib/log"
	"github.com/snowzach/quectool/quectool/atserver"
	cli "github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(atCmd)
	atCmd.PersistentFlags().StringVarP(&atCmdPort, "port", "p", defaults()["modem.port"].(string), "Modem Port")
}

var (
	atCmdPort string
	atCmd     = &cli.Command{
		Use:   "atcmd",
		Short: "Send AT Command",
		Long:  `Start Server`,
		RunE: func(cmd *cli.Command, args []string) error { // Initialize the databse

			if len(args) == 0 {
				return errors.New("no command provided")
			}

			atserver, err := atserver.NewATServer(atCmdPort, conf.C.Duration("modem.timeout"))
			if err != nil {
				log.Fatalf("could not create AT server: %v", err)
			}

			result, err := atserver.SendCMD(context.Background(), args[0])
			if err != nil {
				log.Fatalf("could not send command: %v", err)
			}

			for _, line := range result.Response {
				fmt.Println(line)
			}
			fmt.Println(result.Status.String())

			return nil
		},
	}
)
