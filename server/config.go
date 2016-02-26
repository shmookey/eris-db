package server

import (
	"github.com/shmookey/eris-db/Godeps/_workspace/src/github.com/naoina/toml"
	"github.com/shmookey/eris-db/files"
)

// Standard configuration file for the server.
type (
	ServerConfig struct {
		Bind      Bind      `toml:"bind"`
		TLS       TLS       `toml:"TLS"`
		CORS      CORS      `toml:"CORS"`
		HTTP      HTTP      `toml:"HTTP"`
		WebSocket WebSocket `toml:"web_socket"`
		Logging   Logging   `toml:"logging"`
	}

	Bind struct {
		Address string `toml:"address"`
		Port    uint16 `toml:"port"`
	}

	TLS struct {
		TLS      bool   `toml:"tls"`
		CertPath string `toml:"cert_path"`
		KeyPath  string `toml:"key_path"`
	}

	// Options stores configurations
	CORS struct {
		Enable           bool     `toml:"enable"`
		AllowOrigins     []string `toml:"allow_origins"`
		AllowCredentials bool     `toml:"allow_credentials"`
		AllowMethods     []string `toml:"allow_methods"`
		AllowHeaders     []string `toml:"allow_headers"`
		ExposeHeaders    []string `toml:"expose_headers"`
		MaxAge           uint64   `toml:"max_age"`
	}

	HTTP struct {
		JsonRpcEndpoint string `toml:"json_rpc_endpoint"`
	}

	WebSocket struct {
		WebSocketEndpoint    string `toml:"websocket_endpoint"`
		MaxWebSocketSessions uint   `toml:"max_websocket_sessions"`
		ReadBufferSize       uint   `toml:"read_buffer_size"`
		WriteBufferSize      uint   `toml:"write_buffer_size"`
	}

	Logging struct {
		ConsoleLogLevel string `toml:"console_log_level"`
		FileLogLevel    string `toml:"file_log_level"`
		LogFile         string `toml:"log_file"`
	}
)

func DefaultServerConfig() *ServerConfig {
	cp := ""
	kp := ""
	return &ServerConfig{
		Bind: Bind{
			Address: "",
			Port:    1337,
		},
		TLS: TLS{TLS: false,
			CertPath: cp,
			KeyPath:  kp,
		},
		CORS: CORS{},
		HTTP: HTTP{JsonRpcEndpoint: "/rpc"},
		WebSocket: WebSocket{
			WebSocketEndpoint:    "/socketrpc",
			MaxWebSocketSessions: 50,
			ReadBufferSize:       4096,
			WriteBufferSize:      4096,
		},
		Logging: Logging{
			ConsoleLogLevel: "info",
			FileLogLevel:    "warn",
			LogFile:         "",
		},
	}
}

// Read a TOML server configuration file.
func ReadServerConfig(filePath string) (*ServerConfig, error) {
	bts, err := files.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	cfg := &ServerConfig{}
	err2 := toml.Unmarshal(bts, cfg)
	if err2 != nil {
		return nil, err2
	}
	return cfg, nil
}

// Write a server configuration file.
func WriteServerConfig(filePath string, cfg *ServerConfig) error {
	bts, err := toml.Marshal(*cfg)
	if err != nil {
		return err
	}
	return files.WriteAndBackup(filePath, bts)
}
