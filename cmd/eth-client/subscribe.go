package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	backend "github.com/oasislabs/oasis-gateway/backend/core"
	"github.com/spf13/cobra"
)

type SubscribeProps struct {
	ClientProps ClientProps
	Request     backend.CreateSubscriptionRequest
}

func runSubscribe(props SubscribeProps) error {
	client, err := dialClient(props.ClientProps)
	if err != nil {
		return err
	}

	ch := make(chan interface{}, 64)
	if err := client.SubscribeRequest(context.Background(), props.Request, ch); err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	for ev := range ch {
		if err := encoder.Encode(ev); err != nil {
			fmt.Println("failed to serialize event to json: ", err)
			continue
		}
	}

	return nil
}

func bindSubscribe(cmd *cobra.Command) {
	var props SubscribeProps

	var subscribeCmd = &cobra.Command{
		Use:   "subscribe",
		Short: "Subscribe to events",
		Long: `Subscribe to events with a WS connection to a web3 compatible
server.`,
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if err := runSubscribe(props); err != nil {
				fmt.Println("ERROR: ", err)
				os.Exit(1)
			}
		},
	}

	subscribeCmd.PersistentFlags().StringVar(&props.ClientProps.PrivateKey, "privateKey", "", "the hex encoded wallet's private key")
	subscribeCmd.PersistentFlags().StringVar(&props.ClientProps.URL, "url", "", "the websocket endpoint to the web3 server")
	subscribeCmd.PersistentFlags().StringVar(&props.Request.Event, "event", "", "event type to subscribe to")
	subscribeCmd.PersistentFlags().StringVar(&props.Request.Address, "address", "", "service's address")
	subscribeCmd.PersistentFlags().StringVar(&props.Request.SubID, "subid", "subscription", "subscription id set by the client. "+
		"It is an optional value that should not affect the behavour fo the client in any way")

	cmd.AddCommand(subscribeCmd)
}
