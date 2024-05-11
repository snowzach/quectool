package cmd

import (
	"errors"
	"fmt"

	"github.com/snowzach/golib/log"
	cli "github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	rootCmd.AddCommand(passwdCmd)
}

var (
	passwdCmd = &cli.Command{
		Use:   "passwd",
		Short: "Generate bcyrpt hash of password",
		Long:  `Generate bcyrpt hash of password`,
		RunE: func(cmd *cli.Command, args []string) error { // Initialize the databse

			if len(args) != 1 {
				return errors.New("no password provided")
			}

			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(args[0]), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("could not generate password: %v", err)
			}

			fmt.Println(string(hashedPassword))

			return nil
		},
	}
)
