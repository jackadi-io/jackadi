package task

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/jackadi-io/jackadi/cmd/jack/autocompletion"
	"github.com/jackadi-io/jackadi/cmd/jack/connection"
	"github.com/jackadi-io/jackadi/cmd/jack/option"
	"github.com/jackadi-io/jackadi/cmd/jack/style"
	"github.com/jackadi-io/jackadi/cmd/jack/subcommand/agent"
	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/helper"
	"github.com/jackadi-io/jackadi/internal/parser"
	"github.com/jackadi-io/jackadi/internal/proto"
	"github.com/jackadi-io/jackadi/internal/serializer"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// parseLockMode converts a string lock mode to proto.LockMode.
func parseLockMode(lockMode string) proto.LockMode {
	switch strings.ToLower(strings.TrimSpace(lockMode)) {
	case "none":
		return proto.LockMode_NO_LOCK
	case "write":
		return proto.LockMode_WRITE
	case "exclusive":
		return proto.LockMode_EXCLUSIVE
	case "default":
		return proto.LockMode_UNSPECIFIED
	default:
		return proto.LockMode_UNSPECIFIED
	}
}

type proxyResponse struct {
	*proto.TaskResponse
	Output string `json:"output"` // in the original TaskResponse, Output is a []byte
}

func RunCommand() *cobra.Command {
	target := Target{}
	timeout := int(config.DefaultTaskTimeout.Seconds())
	lockMode := "no-lock"

	cmd := &cobra.Command{
		Use:   "run [ -t | -l | -g | -e | -f ] TARGET COLLECTION:TASK -- ARGS...",
		Short: "Run a task on one or multiple agents",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				err := fmt.Errorf("requires at least %d arg(s), only received %d", 2, len(args))
				fmt.Println(style.RenderError(err.Error()))
				_ = cmd.Help()
				os.Exit(1)
			}
			if args[0] == "" {
				err := errors.New("target must not be empty")
				fmt.Println(style.RenderError(err.Error()))
				_ = cmd.Help()
				os.Exit(1)
			}
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0:
				// agent name completion
				return agent.ListAgent(), cobra.ShellCompDirectiveNoFileComp
			case 1:
				// collection:task completion using plugin in plugin directory + builtins
				return autocompletion.GetCollectionTaskCompletions(toComplete)
			default:
				return []string{}, cobra.ShellCompDirectiveNoFileComp
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			targets := args[0]
			if target.File {
				var err error
				targets, err = targetsFromFile(args[0])
				if err != nil {
					fmt.Println(style.RenderError(err.Error()))
					os.Exit(1)
				}
			}

			protoLockMode := parseLockMode(lockMode)
			out, err := sendTask(targets, target.Mode(), protoLockMode, timeout, args[1], args[2:]...)
			if err != nil {
				e := status.Convert(err)
				fmt.Println(style.RenderError(e.Message()))
				os.Exit(1)
			}

			if option.GetJSONFormat() {
				decodedResponses := make(map[string]*proxyResponse)

				for agentName, response := range out.GetResponses() {
					decodedResponse := proxyResponse{
						TaskResponse: response,
						Output:       string(response.Output), // Decode bytes to string
					}
					decodedResponses[agentName] = &decodedResponse
				}

				result, err := serializer.JSON.MarshalIndent(decodedResponses, "", "  ")
				if err != nil {
					fmt.Println(style.RenderError(fmt.Sprintf("failed to serialize response in JSON: %s", err)))
					os.Exit(1)
				}
				fmt.Println(string(result))
			} else {
				printTaskResult(out)
			}
		},
		GroupID: "operations",
	}

	cmd.Flags().BoolVarP(&target.Exact, "target", "t", false, "target a specific agent")
	cmd.Flags().BoolVarP(&target.List, "list", "l", false, "target a list of agents, separator: ','")
	cmd.Flags().BoolVarP(&target.File, "file", "f", false, "target a list of agents from a file (one agent per line)")
	cmd.Flags().BoolVarP(&target.Glob, "glob", "g", false, "target agents matching the Glob pattern")
	cmd.Flags().BoolVarP(&target.Regexp, "regexp", "e", false, "target agents matching the regular expression")
	cmd.Flags().BoolVarP(&target.Query, "query", "q", false, "target agents using a query")
	cmd.MarkFlagsMutuallyExclusive("target", "list", "glob", "regexp", "query", "file")

	cmd.Flags().IntVar(&timeout, "timeout", 30, "task timeout in second")
	cmd.Flags().StringVar(&lockMode, "lock-mode", "default", "task lock mode: none (concurrent), write (single writer, allows concurrent readers), exclusive (exclusive lock)")

	// Add shell completion for lock mode flag
	_ = cmd.RegisterFlagCompletionFunc("lock-mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"default\tUse default mode defined by the task",
			"none\tAllow concurrent execution",
			"write\tSingle writer - one write task at a time, allows concurrent readers",
			"exclusive\tExclusive lock - only one task runs at a time",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func targetsFromFile(file string) (string, error) {
	fd, err := os.Open(file)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(fd)
	agents := []string{}
	for scanner.Scan() {
		agent := scanner.Text()
		if slices.Contains(agents, agent) {
			continue
		}
		agents = append(agents, agent)
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(agents, ","), nil
}

func sendTask(target string, targetMode proto.TargetMode, lockMode proto.LockMode, timeout int, task string, args ...string) (*proto.FwdResponse, error) {
	conn, err := connection.DialCLI()
	if err != nil {
		return nil, errors.New("failed to connect to the manager")
	}
	defer conn.Close()

	client := proto.NewForwarderClient(conn)

	// ctxReq timeout is 1 second more than expected timeout to give time to the manager or agent to
	// send a timeout response with the IDs of the task.
	ctxReq, cancel := context.WithTimeout(context.Background(), time.Duration(timeout+1)*time.Second)
	defer cancel()

	arguments, err := parser.ParseArgs(args)
	if err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	argList, err := structpb.NewList(arguments.Positional)
	if err != nil {
		return nil, fmt.Errorf("failed to convert arguments to protobuf list: %w", err)
	}

	opts, err := structpb.NewStruct(arguments.Options)
	if err != nil {
		panic(err)
	}

	input := proto.Input{
		Args:    argList,
		Options: opts,
	}

	responses, err := client.ExecTask(ctxReq, &proto.TaskRequest{
		Target:     target,
		TargetMode: targetMode,
		LockMode:   lockMode,
		Task:       task,
		Input:      &input,
		Timeout:    helper.IntToUint32(timeout), // ctxReq should always be superior to this value
	})

	if err != nil {
		return nil, fmt.Errorf("not sent: %s", status.Convert(err).Message())
	}

	return responses, nil
}
