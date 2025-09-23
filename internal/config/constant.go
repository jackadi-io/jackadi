package config

import "time"

const (
	// Grammar.
	PluginSeparator   = ":"
	ListSeparator     = ","
	SpecManagerPrefix = "specs" // Prefix used for specs-related tasks.
	InstantPingName   = "instant-ping"

	// Network.
	DefaultManagerAddress   = "127.0.0.1"
	DefaultManagerPort      = "40080"
	DefaultPluginServerPort = "40081" // Default port for serving plugins.
	HTTPReadHeaderTimeout   = 10 * time.Second

	PluginServerPath = "/plugin/"                  // Path prefix for plugin server endpoints.
	CLISocket        = "/run/jackadi/manager.sock" // Unix socket path for CLI communication.

	// Timing and duration config.
	TaskTimeout             = 30 * time.Second
	DefaultReconnectDelay   = 10 * time.Second // The default delay between reconnection to the manager attempts.
	GracefulShutdownTimeout = 30 * time.Second
	SpecCollectionInterval  = 1 * time.Minute
	DatabaseGCInterval      = 5 * time.Minute
	AgentRetryDelay         = 10 * time.Second // The delay before retrying agent registration.
	PluginUpdateTimeout     = 30 * time.Second

	// gRPC keepalive settings.
	KeepaliveTime          = 5 * time.Second
	KeepaliveTimeout       = 1 * time.Second
	KeepaliveMinTime       = 5 * time.Second
	ClientKeepaliveTime    = 10 * time.Second
	ClientKeepaliveTimeout = 30 * time.Second

	// Task limits.
	MaxConcurrentTasks = 2
	MaxWaitingRequests = 1000 // Mximum number of requests that can wait in queue.

	// `jack results list` limits.
	ResultsPageLimit = 100 // Maximum number of results per page for pagination.
	ResultsLimit     = 100 // Default number of results returned.
	MaxResultsLimit  = 500 // Maximum number of results that can be requested..

	// File and directory paths.
	DefaultConfigDir      = "/etc/jackadi"              // Default configuration directory.
	DefaultPluginDir      = "/opt/jackadi/plugins"      // Default plugin directory for managers.
	DefaultAgentPluginDir = "/var/lib/jackadi/plugins"  // Default plugin directory for agents.
	DatabaseDir           = "/var/lib/jackadi/database" // Default database directory (containing past job results etc...).
	RegistryFileName      = "registry.json"             // Name of the agent registry file.

	// Database and storage settings.
	DBTaskRequestTTL = 24 * time.Hour // TTL of task requests in the database.
	DBTaskResultTTL  = 24 * time.Hour // TTL of task results in the database.
	DBGCThreshold    = 0.7            // Threshold for database garbage collection.

	// Agent activity and health check settings.
	AgentActiveThreshold   = 60 * time.Second // Time threshold to consider an agent active (more than this value means 'inactive').
	ResponseChannelTimeout = 30 * time.Second // Timeout for sending back responses to requester.
)
