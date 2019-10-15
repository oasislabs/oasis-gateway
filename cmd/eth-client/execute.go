package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	backend "github.com/oasislabs/oasis-gateway/backend/core"
	"github.com/spf13/cobra"
)

type ExecuteProps struct {
	ClientProps ClientProps
	Request     backend.ExecuteServiceRequest
}

func runExecute(props ExecuteProps) error {
	client, err := dialClient(props.ClientProps)
	if err != nil {
		return err
	}

	res, err := client.ExecuteService(context.Background(), 0, props.Request)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(res); err != nil {
		fmt.Println("failed to serialize event to json: ", err)
	}

	return nil
}

func bindExecute(cmd *cobra.Command) {
	var props ExecuteProps

	var deployCmd = &cobra.Command{
		Use:   "execute",
		Short: "execute an action on a service",
		Long:  "Executes a function on a service that has been previously deployed.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if err := runExecute(props); err != nil {
				fmt.Println("ERROR: ", err)
				os.Exit(1)
			}
		},
	}

	deployCmd.PersistentFlags().StringVar(
		&props.ClientProps.PrivateKey, "privateKey", "", "the hex encoded wallet's private key")
	deployCmd.PersistentFlags().StringVar(
		&props.ClientProps.URL, "url", "", "the websocket endpoint to the web3 server")
	deployCmd.PersistentFlags().StringVar(
		&props.Request.Data, "data", "", "transaction data for the deployment")
	deployCmd.PersistentFlags().StringVar(
		&props.Request.Address, "address", "", "address of the service")
	deployCmd.PersistentFlags().StringVar(
		&props.Request.SessionKey, "key", "user", "key is the request issuer. Any non-empty value should work.")

	cmd.AddCommand(deployCmd)
}
