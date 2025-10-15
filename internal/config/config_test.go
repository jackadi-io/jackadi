package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func getProgramName() string {
	if len(os.Args) > 0 {
		return os.Args[0]
	}
	return "test"
}

func setupTestDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func createTestConfigFile(t *testing.T, name, content string) string {
	t.Helper()
	testDir := setupTestDir(t)

	configFile := filepath.Join(testDir, name+".yaml")
	if err := os.WriteFile(configFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(configFile)
	})

	return configFile
}

func createTestManagerConfigFile(t *testing.T, content string) string {
	t.Helper()
	testDir := setupTestDir(t)

	yamlFile := filepath.Join(testDir, "manager.yaml")
	if err := os.WriteFile(yamlFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(yamlFile)
	})

	return filepath.Join(testDir, "manager")
}

func setupAgentTest(t *testing.T, flags map[string]string, envVars map[string]string) {
	t.Helper()

	pflag.CommandLine = pflag.NewFlagSet(getProgramName(), pflag.ExitOnError)
	SetupAgentFlags()

	for flag, value := range flags {
		if err := pflag.Set(flag, value); err != nil {
			t.Fatalf("Failed to set flag %s: %v", flag, err)
		}
	}

	for key, value := range envVars {
		t.Setenv(key, value)
	}
}

func setupManagerTest(t *testing.T, flags map[string]string, envVars map[string]string) {
	t.Helper()

	pflag.CommandLine = pflag.NewFlagSet(getProgramName(), pflag.ExitOnError)
	SetupManagerFlags()

	for flag, value := range flags {
		if err := pflag.Set(flag, value); err != nil {
			t.Fatalf("Failed to set flag %s: %v", flag, err)
		}
	}

	for key, value := range envVars {
		t.Setenv(key, value)
	}
}

func TestLoadAgentConfig_Default(t *testing.T) {
	// Create a temporary directory for the plugin directory to avoid permission issues
	tempPluginDir := filepath.Join(t.TempDir(), "plugins")

	setupAgentTest(t, map[string]string{
		"plugin-dir": tempPluginDir,
	}, nil)

	got, err := LoadAgentConfig("")
	if err != nil {
		t.Fatalf("LoadAgentConfig() error = %v", err)
	}

	hostname, _ := os.Hostname()
	expected := &AgentConfig{
		AgentID:          hostname,
		ManagerAddress:   DefaultManagerAddress,
		ManagerPort:      DefaultManagerPort,
		ReconnectDelay:   int(DefaultReconnectDelay.Seconds()),
		PluginDir:        tempPluginDir,
		PluginServerPort: DefaultPluginServerPort,
		CustomResolvers:  []string{},
		MTLS: MTLSConfig{
			Enabled:   true,
			Key:       "",
			Cert:      "",
			ManagerCA: "",
		},
	}

	if diff := cmp.Diff(got, expected); diff != "" {
		t.Errorf("Config mismatch:\n%s", diff)
	}

	if _, err := os.Stat(got.PluginDir); os.IsNotExist(err) {
		t.Errorf("Plugin directory %s was not created", got.PluginDir)
	}
}

func TestLoadAgentConfig_Full(t *testing.T) {
	content := `
agent-id: "full-agent"
manager-address: "192.168.1.1"
manager-port: "8080"
reconnect-delay: 15
plugin-dir: "/tmp/agent-plugins"
plugin-server-port: "8081"
custom-resolvers:
  - "8.8.8.8"
  - "1.1.1.1"
mtls:
  enabled: true
  key: "/path/to/agent.key"
  cert: "/path/to/agent.cert"
  manager-ca-cert: "/path/to/manager-ca.cert"
`

	configFile := createTestConfigFile(t, "agent-full", content)
	setupAgentTest(t, nil, nil)

	got, err := LoadAgentConfig(configFile)
	if err != nil {
		t.Fatalf("LoadAgentConfig() error = %v", err)
	}

	expected := &AgentConfig{
		AgentID:          "full-agent",
		ManagerAddress:   "192.168.1.1",
		ManagerPort:      "8080",
		ReconnectDelay:   15,
		PluginDir:        "/tmp/agent-plugins",
		PluginServerPort: "8081",
		CustomResolvers:  []string{"8.8.8.8", "1.1.1.1"},
		MTLS: MTLSConfig{
			Enabled:   true,
			Key:       "/path/to/agent.key",
			Cert:      "/path/to/agent.cert",
			ManagerCA: "/path/to/manager-ca.cert",
		},
	}

	if diff := cmp.Diff(got, expected); diff != "" {
		t.Errorf("Config mismatch:\n%s", diff)
	}
}

func TestLoadManagerConfig_Default(t *testing.T) {
	setupManagerTest(t, nil, nil)

	got, err := LoadManagerConfig("")
	if err != nil {
		t.Fatalf("LoadManagerConfig() error = %v", err)
	}

	expected := &ManagerConfig{
		ManagerID:        "",
		ConfigDir:        DefaultConfigDir,
		ListenAddress:    DefaultManagerAddress,
		ListenPort:       DefaultManagerPort,
		PluginDir:        DefaultPluginDir,
		PluginServerPort: DefaultPluginServerPort,
		AutoAcceptAgent:  false,
		MTLS: ManagerMTLSConfig{
			Enabled: true,
			Key:     "",
			Cert:    "",
			AgentCA: "",
		},
		API: APIConfig{
			Enabled: true,
			Address: DefaultAPIAddress,
			Port:    DefaultAPIPort,
			TLS: APITLSConfig{
				Enabled: false,
				Cert:    "",
				Key:     "",
			},
		},
	}

	if diff := cmp.Diff(got, expected); diff != "" {
		t.Errorf("Config mismatch:\n%s", diff)
	}
}

func TestLoadManagerConfig_Full(t *testing.T) {
	content := `
manager-id: "full-manager"
config-dir: "/etc/full-config"
address: "0.0.0.0"
port: "9090"
plugin-dir: "/opt/full-plugins"
plugin-server-port: "9091"
auto-accept-agent: true
mtls:
  enabled: true
  key: "/path/to/manager.key"
  cert: "/path/to/manager.cert"
  agent-ca-cert: "/path/to/agent-ca.cert"
api:
  enabled: true
  address: "127.0.0.1"
  port: "8080"
  tls:
    enabled: true
    cert: "/path/to/api.cert"
    key: "/path/to/api.key"
`

	configFile := createTestManagerConfigFile(t, content)
	setupManagerTest(t, nil, nil)

	got, err := LoadManagerConfig(configFile)
	if err != nil {
		t.Fatalf("LoadManagerConfig() error = %v", err)
	}

	expected := &ManagerConfig{
		ManagerID:        "full-manager",
		ConfigDir:        "/etc/full-config",
		ListenAddress:    "0.0.0.0",
		ListenPort:       "9090",
		PluginDir:        "/opt/full-plugins",
		PluginServerPort: "9091",
		AutoAcceptAgent:  true,
		MTLS: ManagerMTLSConfig{
			Enabled: true,
			Key:     "/path/to/manager.key",
			Cert:    "/path/to/manager.cert",
			AgentCA: "/path/to/agent-ca.cert",
		},
		API: APIConfig{
			Enabled: true,
			Address: "127.0.0.1",
			Port:    "8080",
			TLS: APITLSConfig{
				Enabled: true,
				Cert:    "/path/to/api.cert",
				Key:     "/path/to/api.key",
			},
		},
	}

	if diff := cmp.Diff(got, expected); diff != "" {
		t.Errorf("Config mismatch:\n%s", diff)
	}
}

func TestSetupAgentFlags(t *testing.T) {
	pflag.CommandLine = pflag.NewFlagSet(getProgramName(), pflag.ExitOnError)
	SetupAgentFlags()

	expectedFlags := []string{
		"id", "manager-address", "manager-port", "reconnect-delay",
		"plugin-dir", "plugin-server-port", "custom-resolvers",
		"mtls.enabled", "mtls.key", "mtls.cert", "mtls.manager-ca-cert",
		"config",
	}

	for _, flagName := range expectedFlags {
		if flag := pflag.Lookup(flagName); flag == nil {
			t.Errorf("Expected flag %s to be defined", flagName)
		}
	}
}

func TestSetupManagerFlags(t *testing.T) {
	pflag.CommandLine = pflag.NewFlagSet(getProgramName(), pflag.ExitOnError)
	SetupManagerFlags()

	expectedFlags := []string{
		"id", "config-dir", "address", "port", "plugin-dir", "plugin-server-port",
		"auto-accept-agent", "mtls.enabled", "mtls.key", "mtls.cert",
		"mtls.agent-ca-cert", "api.enabled", "api.address", "api.port",
		"api.tls.enabled", "api.tls.cert", "api.tls.key", "config",
	}

	for _, flagName := range expectedFlags {
		if flag := pflag.Lookup(flagName); flag == nil {
			t.Errorf("Expected flag %s to be defined", flagName)
		}
	}
}

func TestGetConfigFile(t *testing.T) {
	tests := []struct {
		name       string
		configFlag string
		want       string
	}{
		{
			name:       "config flag set",
			configFlag: "/path/to/config.yaml",
			want:       "/path/to/config.yaml",
		},
		{
			name:       "config flag empty",
			configFlag: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pflag.CommandLine = pflag.NewFlagSet(getProgramName(), pflag.ExitOnError)
			pflag.String("config", "", "config file path")

			if tt.configFlag != "" {
				_ = pflag.Set("config", tt.configFlag)
			}

			v := viper.New()
			if err := v.BindPFlags(pflag.CommandLine); err != nil {
				t.Fatalf("Failed to bind flags: %v", err)
			}

			got := GetConfigFile(v)
			if got != tt.want {
				t.Errorf("GetConfigFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
