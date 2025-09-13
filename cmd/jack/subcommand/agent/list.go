package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackadi-io/jackadi/cmd/jack/connection"
	"github.com/jackadi-io/jackadi/cmd/jack/option"
	"github.com/jackadi-io/jackadi/cmd/jack/style"
	"github.com/jackadi-io/jackadi/internal/proto"
	"github.com/jackadi-io/jackadi/internal/serializer"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/status"
)

func healthCommand() *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:   "health [OPTION] ...",
		Short: "agents health",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := list(0)
			if err != nil {
				r := status.Convert(err)
				fmt.Println(r.Message())
				os.Exit(1)
			}

			if option.GetJSONFormat() {
				result, err := serializer.JSON.MarshalIndent(resp, "", "   ")
				if err != nil {
					fmt.Println("failed to serialize response in JSON: %w", err)
					os.Exit(1)
				}
				fmt.Println(string(result))
			} else {
				in := style.Title("Agents")
				in += prettyAgentsHealthSprint(resp.Accepted, verbose)

				style.PrettyPrint(in)
			}
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show agent details")

	return cmd
}

func listCommand() *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:   "list [OPTION] ...",
		Short: "list agents",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := list(0)
			if err != nil {
				r := status.Convert(err)
				fmt.Println(r.Message())
				os.Exit(1)
			}

			if option.GetJSONFormat() {
				result, err := serializer.JSON.MarshalIndent(resp, "", "   ")
				if err != nil {
					fmt.Println("failed to serialize response in JSON: %w", err)
					os.Exit(1)
				}
				fmt.Println(string(result))
			} else {
				in := style.Title("Accepted")
				in += prettyAgentListSprint(resp.Accepted, verbose)
				in += style.Title("Candidates")
				in += prettyAgentListSprint(resp.Candidates, verbose)
				in += style.Title("Rejected")
				in += prettyAgentListSprint(resp.Rejected, verbose)

				style.PrettyPrint(in)
			}
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show agent details")

	return cmd
}

func list(filter proto.Filter) (*proto.ListAgentsResponse, error) {
	conn, err := connection.DialCLI()
	if err != nil {
		return nil, errors.New("failed to connect the manager")
	}
	defer conn.Close()
	client := proto.NewAPIClient(conn)

	ctxReq, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	resp, err := client.ListAgents(ctxReq, &proto.ListAgentsRequest{
		Filter: filter,
	})
	if err != nil {
		return nil, errors.New("failed to get list of agents")
	}

	return resp, err
}

func ListAgent() []string {
	resp, err := list(proto.Filter_NONE)
	if err != nil {
		return nil
	}

	hosts := []string{}
	for _, a := range resp.GetAccepted() {
		hosts = append(hosts, a.GetId())
	}
	return hosts
}
