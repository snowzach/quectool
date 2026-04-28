package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	cli "github.com/spf13/cobra"

	"github.com/snowzach/quectool/quectool/tlsgen"
)

func init() {
	gencertCmd.Flags().StringVar(&gencertCertOut, "cert", "server.crt", "output path for the certificate (PEM)")
	gencertCmd.Flags().StringVar(&gencertKeyOut, "key", "server.key", "output path for the private key (PEM)")
	gencertCmd.Flags().StringSliceVar(&gencertHosts, "host", nil,
		"DNS name or IP to include in the certificate's SAN (repeatable). "+
			"If unset, all the host's interface IPs plus 'localhost' are used.")
	gencertCmd.Flags().DurationVar(&gencertValidity, "validity", 10*365*24*time.Hour,
		"how long the certificate should be valid for")
	gencertCmd.Flags().BoolVar(&gencertForce, "force", false,
		"overwrite existing cert/key files instead of refusing")
	rootCmd.AddCommand(gencertCmd)
}

var (
	gencertCertOut  string
	gencertKeyOut   string
	gencertHosts    []string
	gencertValidity time.Duration
	gencertForce    bool

	gencertCmd = &cli.Command{
		Use:   "gencert",
		Short: "Generate a self-signed TLS certificate + key",
		Long: `Generate a long-lived self-signed certificate and private key for
serving the web UI over HTTPS. The cert covers all of the host's interface
IPs by default, so it works whether you reach the modem via its LAN address,
USB-RNDIS, or loopback. Default validity is 10 years.

The server normally generates this on first boot if missing — only call
gencert manually to regenerate after a network reconfiguration (so the SAN
list picks up new IPs).`,
		RunE: runGencert,
	}
)

func runGencert(_ *cli.Command, _ []string) error {
	if !gencertForce {
		for _, p := range []string{gencertCertOut, gencertKeyOut} {
			if _, err := os.Stat(p); err == nil {
				return fmt.Errorf("%s exists; pass --force to overwrite", p)
			}
		}
	}
	res, err := tlsgen.Generate(tlsgen.Config{
		CertPath: gencertCertOut,
		KeyPath:  gencertKeyOut,
		Hosts:    gencertHosts,
		Validity: gencertValidity,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s (cert) and %s (key)\n", gencertCertOut, gencertKeyOut)
	fmt.Fprintf(os.Stderr, "  hosts: %s\n", strings.Join(res.Hosts, ", "))
	fmt.Fprintf(os.Stderr, "  valid until: %s\n", res.NotAfter.Format(time.RFC3339))
	return nil
}
