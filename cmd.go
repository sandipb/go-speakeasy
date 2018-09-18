package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	hostname            string
	metricsSocket       = "/var/tmp/metrics_socket"
	commandPort         = "127.0.0.1:26300"
	publishPort         string
	emitterName         = "simple"
	emitterArgs         = []string{}
	emitIntervalSeconds = 60
	socketMode          = ""
	logLevelName        = "info"
)

func init() {
	mainCmd.PersistentFlags().StringVarP(&metricsSocket, "metrics-socket", "s", metricsSocket, "Metrics socket path where server listens to")
	mainCmd.PersistentFlags().StringVarP(&logLevelName, "log-level", "l", "info", "Set log level")

	serverCmd.PersistentFlags().StringVarP(&hostname, "hostname", "H", "", "Source hostname to use while emitting metrics")
	serverCmd.Flags().StringVarP(&socketMode, "socket-mode", "m", "", "Permissions in octal for the metrics socket")
	serverCmd.Flags().StringVarP(&commandPort, "command-port", "C", commandPort, "Port for admin commands")
	serverCmd.Flags().StringVarP(&publishPort, "publish-port", "P", "", "PUB port for event notifications")

	serverCmd.Flags().StringVarP(&emitterName, "emitter", "e", "simple", "Emitter to use")
	serverCmd.Flags().StringArrayVarP(&emitterArgs, "emitter-args", "E", []string{}, "Arguments to the emitter in the form key=val. Can be repeated")
	serverCmd.Flags().IntVarP(&emitIntervalSeconds, "emit-interval", "i", emitIntervalSeconds, "Interval in seconds to emit metrics")

	mainCmd.AddCommand(serverCmd)
	mainCmd.AddCommand(clientCmd)
}

// preRun runs before each command
func preRun(cmd *cobra.Command, args []string) {
	if lvl, err := log.ParseLevel(logLevelName); err != nil {
		logger.Fatalf("Invalid log level: %s", lvl)
	} else {
		logger.SetLevel(lvl)
	}

}

var mainCmd = &cobra.Command{
	Use:   "speakeasy",
	Short: "Speakeasy is a host level metrics aggregator and emitter",
}

var serverCmd = &cobra.Command{
	Use:    "server",
	Short:  "Set up a server to receive metrics and emit them",
	PreRun: preRun,
	Run:    runServerCommand,
}

var clientCmd = &cobra.Command{
	Use:    "client",
	Short:  "Send metrics to a speakeasy server",
	PreRun: preRun,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Start client")
	},
}
