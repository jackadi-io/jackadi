package builtin

import (
	"log"
	"log/slog"

	"github.com/jackadi-io/jackadi/internal/config"
	"github.com/jackadi-io/jackadi/internal/plugin/inventory"
	"github.com/jackadi-io/jackadi/sdk"
)

func ping() (bool, error) {
	return true, nil
}

func MustLoadHealth() {
	cmd := sdk.New("health")

	// health:instant-ping is routed earlier in the agent listener.
	cmd.MustRegisterTask(config.InstantPingName, ping).
		WithSummary("Execute immediate ping.").
		WithDescription("This healthcheck bypasses any tasks queue and lock. The goal is to check gRPC connectivity only.")

	cmd.MustRegisterTask("ping", ping).
		WithSummary("Ping.").
		WithDescription("Normal healthcheck using the task queue.")

	if err := inventory.Registry.Register(cmd); err != nil {
		name, _ := cmd.Name()
		slog.Error("could not load builtin task", "error", err, "task", name)
		log.Fatal(err)
	}
}
