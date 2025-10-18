package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackadi-io/jackadi/cmd/jack/connection"
	"github.com/jackadi-io/jackadi/cmd/jack/style"
	"github.com/jackadi-io/jackadi/internal/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/status"
)

func acceptCommand() *cobra.Command {
	var verbose bool
	var address, certificate string
	cmd := &cobra.Command{
		Use:   "accept AGENT ...",
		Short: "accept agent",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agent := proto.AgentInfo{
				Id: args[0],
			}
			if address != "" {
				agent.Address = &address
			}
			if certificate != "" {
				agent.Certificate = &certificate
			}
			acceptedAgent, err := accept(&agent)
			if err != nil {
				r := status.Convert(err)
				fmt.Fprintln(os.Stderr, r.Message())
				os.Exit(1)
			}

			style.PrettyPrint(fmt.Sprintf("agent registered: %s\n", acceptedAgent.String()))
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show agent details")
	cmd.Flags().StringVar(&address, "address", "", "specify address of the agent to accept")
	cmd.Flags().StringVar(&certificate, "certificate", "", "specify certificate of the agent to accept")

	return cmd
}

func accept(agent *proto.AgentInfo) (*proto.AgentInfo, error) {
	conn, err := connection.DialCLI()
	if err != nil {
		return nil, errors.New("failed to connect the manager")
	}
	defer conn.Close()
	client := proto.NewAPIClient(conn)

	ctxReq, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	acceptedAgent, err := client.AcceptAgent(ctxReq, &proto.AgentRequest{Agent: agent})
	if err != nil {
		return nil, err
	}

	return acceptedAgent.GetAgent(), nil
}
