package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "eth-client"}

	bindDeploy(rootCmd)
	bindExecute(rootCmd)
	bindSubscribe(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("failed to parse command line arguments ", err.Error())
		os.Exit(1)
	}
}
