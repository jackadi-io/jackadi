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

func removeCommand() *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:   "remove AGENT ...",
		Short: "remove agent",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			agent := proto.AgentInfo{
				Id: args[0],
			}
			if err := remove(&agent); err != nil {
				r := status.Convert(err)
				fmt.Println(r.Message())
				os.Exit(1)
			}

			style.PrettyPrint(fmt.Sprintf("agent removed: %s\n", agent.GetId()))
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show agent details")

	return cmd
}

func remove(agent *proto.AgentInfo) error {
	conn, err := connection.DialCLI()
	if err != nil {
		return errors.New("failed to connect the manager")
	}
	defer conn.Close()
	client := proto.NewAPIClient(conn)

	ctxReq, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, err = client.RemoveAgent(ctxReq, &proto.AgentRequest{Agent: agent})
	if err != nil {
		return err
	}

	return nil
}
