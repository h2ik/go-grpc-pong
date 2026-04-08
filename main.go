package main

import (
	"fmt"
	"os"

	"github.com/h2ik/go-grpc-pong/cmd"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-grpc-pong",
	Short: "gRPC ping/pong test tool for Istio cluster-to-cluster validation",
}

func init() {
	rootCmd.AddCommand(cmd.PingCmd)
	rootCmd.AddCommand(cmd.PongCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
