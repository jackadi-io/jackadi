package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sagikazarmark/locafero"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var errViperConfigNotFound viper.ConfigFileNotFoundError

type AgentConfig struct {
	AgentID          string   `mapstructure:"agent-id" yaml:"agent-id"`
	ManagerAddress   string   `mapstructure:"manager-address" yaml:"manager-address"`
	ManagerPort      string   `mapstructure:"manager-port" yaml:"manager-port"`
	ReconnectDelay   int      `mapstructure:"reconnect-delay" yaml:"reconnect-delay"`
	PluginDir        string   `mapstructure:"plugin-dir" yaml:"plugin-dir"`
	PluginServerPort string   `mapstructure:"plugin-server-port" yaml:"plugin-server-port"`
	MTLS             bool     `mapstructure:"mtls" yaml:"mtls"`
	TLSKey           string   `mapstructure:"mtls-key" yaml:"mtls-key"`
	TLSCert          string   `mapstructure:"mtls-cert" yaml:"mtls-cert"`
	TLSManagerCA     string   `mapstructure:"mtls-manager-ca-cert" yaml:"mtls-manager-ca-cert"`
	CustomResolvers  []string `mapstructure:"custom-resolvers" yaml:"custom-resolvers"`
}

type ManagerConfig struct {
	ConfigDir        string `mapstructure:"config-dir" yaml:"config-dir"`
	ListenAddress    string `mapstructure:"address" yaml:"address"`
	ListenPort       string `mapstructure:"port" yaml:"port"`
	PluginDir        string `mapstructure:"plugin-dir" yaml:"plugin-dir"`
	PluginServerPort string `mapstructure:"plugin-server-port" yaml:"plugin-server-port"`
	AutoAcceptAgent  bool   `mapstructure:"auto-accept-agent" yaml:"auto-accept-agent"`
	MTLS             bool   `mapstructure:"mtls" yaml:"mtls"`
	TLSCert          string `mapstructure:"mtls-cert" yaml:"mtls-cert"`
	TLSKey           string `mapstructure:"mtls-key" yaml:"mtls-key"`
	TLSAgentCA       string `mapstructure:"mtls-agent-ca-cert" yaml:"mtls-agent-ca-cert"`
	APIEnabled       bool   `mapstructure:"api-enabled" yaml:"api-enabled"`
	APIAddress       string `mapstructure:"api-address" yaml:"api-address"`
	APIPort          string `mapstructure:"api-port" yaml:"api-port"`
	APITLSEnabled    bool   `mapstructure:"api-tls-enabled" yaml:"api-tls-enabled"`
	APITLSCert       string `mapstructure:"api-tls-cert" yaml:"api-tls-cert"`
	APITLSKey        string `mapstructure:"api-tls-key" yaml:"api-tls-key"`
}

func SetupAgentFlags() {
	pflag.String("id", "", "set agent ID")
	pflag.String("manager-address", DefaultManagerAddress, "set manager address")
	pflag.String("manager-port", DefaultManagerPort, "set manager port")
	pflag.Int("reconnect-delay", int(DefaultReconnectDelay.Seconds()), "delay between reconnect attempts to the manager, in seconds")
	pflag.String("plugin-dir", DefaultAgentPluginDir, "installed plugin directory")
	pflag.String("plugin-server-port", DefaultPluginServerPort, "manager port used to serve plugins")
	pflag.Bool("mtls", true, "secure connection to managers using mTLS, recommended: true")
	pflag.String("mtls-key", "", "agent TLS key filepath")
	pflag.String("mtls-cert", "", "agent TLS certificate filepath")
	pflag.String("mtls-manager-ca-cert", "", "manager TLS certificate filepath")
	pflag.StringSlice("custom-resolvers", []string{}, "custom DNS resolvers for GRPC connections (comma-separated)")
	pflag.String("config", "", "config file path")
}

func SetupManagerFlags() {
	pflag.String("id", "", "set manager ID")
	pflag.String("config-dir", DefaultConfigDir, "configuration directory")
	pflag.String("address", DefaultManagerAddress, "set manager address")
	pflag.String("port", DefaultManagerPort, "set manager port")
	pflag.String("plugin-dir", DefaultPluginDir, "plugin inventory directory")
	pflag.String("plugin-server-port", DefaultPluginServerPort, "set manager port used to serve plugins")
	pflag.Bool("auto-accept-agent", false, "auto accept new agents")
	pflag.Bool("mtls", true, "secure connections to agents using mTLS, recommended: true")
	pflag.String("mtls-key", "", "manager TLS key filepath")
	pflag.String("mtls-cert", "", "manager TLS certificate filepath")
	pflag.String("mtls-agent-ca-cert", "", "agent TLS certificate filepath")
	pflag.Bool("api-enabled", true, "enable HTTP REST API")
	pflag.String("api-address", DefaultAPIAddress, "HTTP API listen address")
	pflag.String("api-port", DefaultAPIPort, "HTTP API listen port")
	pflag.Bool("api-tls-enabled", false, "enable TLS for HTTP REST API")
	pflag.String("api-tls-cert", "", "API TLS certificate filepath")
	pflag.String("api-tls-key", "", "API TLS key filepath")
	pflag.String("config", "", "config file path")
}

func GetConfigFile(v *viper.Viper) string {
	configFile := v.GetString("config")
	if configFile != "" {
		return configFile
	}
	return ""
}

func LoadAgentConfig(configFile string) (*AgentConfig, error) {
	v := viper.New()

	v.SetDefault("agent-id", "")
	v.SetDefault("manager-address", DefaultManagerAddress)
	v.SetDefault("manager-port", DefaultManagerPort)
	v.SetDefault("reconnect-delay", int(DefaultReconnectDelay.Seconds()))
	v.SetDefault("plugin-dir", DefaultAgentPluginDir)
	v.SetDefault("plugin-server-port", DefaultPluginServerPort)
	v.SetDefault("mtls", true)
	v.SetDefault("mtls-key", "")
	v.SetDefault("mtls-cert", "")
	v.SetDefault("mtls-manager-ca-cert", "")
	v.SetDefault("custom-resolvers", []string{})

	v.SetEnvPrefix("JACKADI_AGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("error reading config file %s: %w", configFile, err)
		}
	} else {
		v.SetConfigName("agent")
		v.SetConfigType("yaml")
		v.AddConfigPath("/etc/jackadi/")
		v.AddConfigPath(".")

		if err := v.ReadInConfig(); err != nil {
			if !errors.As(err, &errViperConfigNotFound) {
				return nil, fmt.Errorf("error reading config file: %w", err)
			}
		}
	}

	if err := v.BindPFlags(pflag.CommandLine); err != nil {
		return nil, fmt.Errorf("error binding flags: %w", err)
	}

	var config AgentConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if config.AgentID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("failed to get hostname, please set the agent-id")
		}
		config.AgentID = hostname
	}

	if err := os.MkdirAll(config.PluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to initialize plugin directory '%s': %w", config.PluginDir, err)
	}

	return &config, nil
}

func LoadManagerConfig(configFile string) (*ManagerConfig, error) {
	finder := locafero.Finder{
		Paths: []string{".", "/etc/jackadi"},
		Names: locafero.NameWithExtensions("manager", viper.SupportedExts...),
		Type:  locafero.FileTypeFile,
	}

	if configFile != "" {
		path, file := filepath.Split(configFile)
		finder.Paths = []string{path}
		finder.Names = locafero.NameWithExtensions(file, viper.SupportedExts...)
	}

	v := viper.NewWithOptions(viper.WithFinder(finder))

	v.SetDefault("manager-id", "")
	v.SetDefault("config-dir", DefaultConfigDir)
	v.SetDefault("address", DefaultManagerAddress)
	v.SetDefault("port", DefaultManagerPort)
	v.SetDefault("plugin-dir", DefaultPluginDir)
	v.SetDefault("plugin-server-port", DefaultPluginServerPort)
	v.SetDefault("auto-accept-agent", false)
	v.SetDefault("mtls", true)
	v.SetDefault("mtls-cert", "")
	v.SetDefault("mtls-key", "")
	v.SetDefault("mtls-agent-ca-cert", "")
	v.SetDefault("api-enabled", true)
	v.SetDefault("api-address", DefaultAPIAddress)
	v.SetDefault("api-port", DefaultAPIPort)
	v.SetDefault("api-tls-enabled", false)
	v.SetDefault("api-tls-cert", "")
	v.SetDefault("api-tls-key", "")

	v.SetEnvPrefix("JACKADI_MANAGER")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if !errors.As(err, &errViperConfigNotFound) {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	if err := v.BindPFlags(pflag.CommandLine); err != nil {
		return nil, fmt.Errorf("error binding flags: %w", err)
	}

	var config ManagerConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}
