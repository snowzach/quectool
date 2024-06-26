package cmd

import (
	"fmt"
	"os"

	cli "github.com/spf13/cobra"

	"github.com/snowzach/golib/conf"
	"github.com/snowzach/golib/log"
	"github.com/snowzach/golib/version"
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
}

var (

	// Config and global logger
	pidFile string
	cfgFile string

	// The Root Cli Handler
	rootCmd = &cli.Command{
		Version: version.GitVersion,
		Use:     version.Executable,
		PersistentPreRunE: func(cmd *cli.Command, args []string) error {

			// Parse defaults, config file and environment.
			if err := conf.C.Parse(
				conf.WithMap(defaults()),
				conf.WithFile(cfgFile),
				conf.WithEnv(),
			); err != nil {
				fmt.Printf("could not load config: %v", err)
				os.Exit(1)
			}

			var loggerConfig log.LoggerConfig
			if err := conf.C.Unmarshal(&loggerConfig, conf.UnmarshalConf{Path: "logger"}); err != nil {
				fmt.Printf("could not parse logger config: %v", err)
				os.Exit(1)
			}
			if err := log.InitLogger(&loggerConfig); err != nil {
				fmt.Printf("could not configure logger: %v", err)
				os.Exit(1)
			}

			// Create Pid File
			pidFile = conf.C.String("pidfile")
			if pidFile != "" {
				file, err := os.OpenFile(pidFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
				if err != nil {
					return fmt.Errorf("could not create pid file: %s error:%v", pidFile, err)
				}
				defer file.Close()
				_, err = fmt.Fprintf(file, "%d\n", os.Getpid())
				if err != nil {
					return fmt.Errorf("could not create pid file: %s error:%v", pidFile, err)
				}
			}
			return nil
		},
		PersistentPostRun: func(cmd *cli.Command, args []string) {
			// Remove Pid file
			if pidFile != "" {
				os.Remove(pidFile)
			}
		},
	}
)

// Execute starts the program
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}
