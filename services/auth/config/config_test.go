package config

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabase_URL(t *testing.T) {
	tests := []struct {
		name     string
		db       *Database
		expected string
	}{
		{
			name: "with user and password",
			db: &Database{
				User:     "testuser",
				password: "testpass",
				Host:     "localhost",
				Port:     "5432",
				Name:     "testdb",
			},
			expected: "postgres://testuser:testpass@localhost:5432/testdb",
		},
		{
			name: "with user only",
			db: &Database{
				User:     "testuser",
				password: "",
				Host:     "localhost",
				Port:     "5432",
				Name:     "testdb",
			},
			expected: "postgres://testuser@localhost:5432/testdb",
		},
		{
			name: "without credentials",
			db: &Database{
				User:     "",
				password: "",
				Host:     "localhost",
				Port:     "5432",
				Name:     "testdb",
			},
			expected: "postgres://localhost:5432/testdb",
		},
		{
			name: "with special characters in password",
			db: &Database{
				User:     "user",
				password: "p@ss:word",
				Host:     "db.example.com",
				Port:     "5432",
				Name:     "mydb",
			},
			expected: "postgres://user:p%40ss%3Aword@db.example.com:5432/mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.db.URL()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLoad_Success(t *testing.T) {
	// Save and restore original getConfigFS
	origFS := getConfigFS
	defer func() { getConfigFS = origFS }()

	// Mock filesystem with config files
	getConfigFS = func() fs.FS {
		return fstest.MapFS{
			"configs/default.toml": &fstest.MapFile{
				Data: []byte(`app-name = "TestApp"
port = "8080"

[database]
user = "rcv"
host = "localhost"
port = "5432"
name = "testdb"
`),
			},
			"configs/testing.toml": &fstest.MapFile{
				Data: []byte(`port = "9090"

[database]
host = "test-host"
`),
			},
		}
	}

	ctx := context.Background()
	envs := map[string]string{
		"DB_PASSWORD": "secret123",
		"APP_ENV":     "testing",
		"APP_NAME":    "TestApp",
	}

	config, err := Load(ctx, envs)

	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.Database)

	// Verify config was loaded and merged
	assert.Equal(t, "TestApp", config.AppName)
	assert.Equal(t, "9090", config.Port) // From testing.toml
	assert.Equal(t, EnvironmentTesting, config.Environment)

	// Verify database config
	assert.Equal(t, "rcv", config.Database.User)
	assert.Equal(t, "test-host", config.Database.Host) // From testing.toml
	assert.Equal(t, "5432", config.Database.Port)
	assert.Equal(t, "testdb", config.Database.Name)

	// Verify URL generation includes password
	url := config.Database.URL()
	assert.Contains(t, url, "secret123")
	assert.Contains(t, url, "test-host")
}

func TestLoad_MissingPasswordEnv(t *testing.T) {
	// Save and restore original getConfigFS
	origFS := getConfigFS
	defer func() { getConfigFS = origFS }()

	// Mock filesystem
	getConfigFS = func() fs.FS {
		return fstest.MapFS{
			"configs/default.toml": &fstest.MapFile{
				Data: []byte(`[database]
user = "rcv"
host = "localhost"
port = "5432"
name = "testdb"
`),
			},
		}
	}

	ctx := context.Background()
	envs := map[string]string{
		"APP_ENV": "development",
		// DB_PASSWORD is missing
	}

	config, err := Load(ctx, envs)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "DB_PASSWORD")
}

func TestLoad_WithDefaultConfigOnly(t *testing.T) {
	// Save and restore original getConfigFS
	origFS := getConfigFS
	defer func() { getConfigFS = origFS }()

	// Mock filesystem with only default config
	getConfigFS = func() fs.FS {
		return fstest.MapFS{
			"configs/default.toml": &fstest.MapFile{
				Data: []byte(`app-name = "DefaultApp"
port = "8080"

[database]
user = "admin"
host = "localhost"
port = "5432"
name = "defaultdb"
`),
			},
		}
	}

	ctx := context.Background()
	envs := map[string]string{
		"DB_PASSWORD": "password",
		"APP_ENV":     "production", // No production.toml exists
		"APP_NAME":    "DefaultApp",
	}

	config, err := Load(ctx, envs)

	require.NoError(t, err)
	require.NotNil(t, config)

	// Should use default config values
	assert.Equal(t, "DefaultApp", config.AppName)
	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, EnvironmentProduction, config.Environment)
}

func TestConfig_GetAppEnv(t *testing.T) {
	config := &Config{
		Environment: EnvironmentStaging,
	}

	env := config.GetAppEnv()
	assert.NotNil(t, env)
	assert.Equal(t, "staging", env.String())
}

func TestPort_String(t *testing.T) {
	tests := []struct {
		name     string
		port     Port
		expected string
	}{
		{
			name:     "standard port 80",
			port:     80,
			expected: ":80",
		},
		{
			name:     "standard port 443",
			port:     443,
			expected: ":443",
		},
		{
			name:     "custom port 8080",
			port:     8080,
			expected: ":8080",
		},
		{
			name:     "port 0",
			port:     0,
			expected: ":0",
		},
		{
			name:     "negative port",
			port:     -1,
			expected: ":-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.port.String()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestServer_URL(t *testing.T) {
	tests := []struct {
		name     string
		server   Server
		path     string
		queries  map[string]string
		expected string
	}{
		{
			name: "http with default port 80 - port omitted",
			server: Server{
				Scheme: "http",
				Host:   "example.com",
				Port:   80,
			},
			path:     "/api/users",
			queries:  nil,
			expected: "http://example.com/api/users",
		},
		{
			name: "https with default port 443 - port omitted",
			server: Server{
				Scheme: "https",
				Host:   "example.com",
				Port:   443,
			},
			path:     "/api/users",
			queries:  nil,
			expected: "https://example.com/api/users",
		},
		{
			name: "http with custom port 8080",
			server: Server{
				Scheme: "http",
				Host:   "localhost",
				Port:   8080,
			},
			path:     "/health",
			queries:  nil,
			expected: "http://localhost:8080/health",
		},
		{
			name: "https with custom port 8443",
			server: Server{
				Scheme: "https",
				Host:   "api.example.com",
				Port:   8443,
			},
			path:     "/v1/resource",
			queries:  nil,
			expected: "https://api.example.com:8443/v1/resource",
		},
		{
			name: "with single query parameter",
			server: Server{
				Scheme: "https",
				Host:   "example.com",
				Port:   443,
			},
			path: "/search",
			queries: map[string]string{
				"q": "test",
			},
			expected: "https://example.com/search?q=test",
		},
		{
			name: "with multiple query parameters",
			server: Server{
				Scheme: "http",
				Host:   "localhost",
				Port:   3000,
			},
			path: "/api/items",
			queries: map[string]string{
				"page":  "1",
				"limit": "10",
				"sort":  "name",
			},
			expected: "http://localhost:3000/api/items?limit=10&page=1&sort=name",
		},
		{
			name: "empty path with queries",
			server: Server{
				Scheme: "https",
				Host:   "example.com",
				Port:   443,
			},
			path: "",
			queries: map[string]string{
				"key": "value",
			},
			expected: "https://example.com?key=value",
		},
		{
			name: "special characters in query values",
			server: Server{
				Scheme: "https",
				Host:   "api.example.com",
				Port:   443,
			},
			path: "/search",
			queries: map[string]string{
				"q": "hello world",
			},
			expected: "https://api.example.com/search?q=hello+world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.server.URL(tt.path, tt.queries)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestEnvironment_IsDev(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		expected bool
	}{
		{
			name:     "development is dev",
			env:      EnvironmentDevelopment,
			expected: true,
		},
		{
			name:     "staging is not dev",
			env:      EnvironmentStaging,
			expected: false,
		},
		{
			name:     "production is not dev",
			env:      EnvironmentProduction,
			expected: false,
		},
		{
			name:     "testing is not dev",
			env:      EnvironmentTesting,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.env.IsDev()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestEnvironment_String(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		expected string
	}{
		{
			name:     "development",
			env:      EnvironmentDevelopment,
			expected: "development",
		},
		{
			name:     "staging",
			env:      EnvironmentStaging,
			expected: "staging",
		},
		{
			name:     "production",
			env:      EnvironmentProduction,
			expected: "production",
		},
		{
			name:     "testing",
			env:      EnvironmentTesting,
			expected: "testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.env.String()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestEnvironment_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		expected bool
	}{
		{
			name:     "valid development",
			env:      EnvironmentDevelopment,
			expected: true,
		},
		{
			name:     "valid staging",
			env:      EnvironmentStaging,
			expected: true,
		},
		{
			name:     "valid production",
			env:      EnvironmentProduction,
			expected: true,
		},
		{
			name:     "valid testing",
			env:      EnvironmentTesting,
			expected: true,
		},
		{
			name:     "invalid environment",
			env:      Environment("invalid"),
			expected: false,
		},
		{
			name:     "empty environment",
			env:      Environment(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.env.IsValid()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  Environment
		shouldErr bool
	}{
		{
			name:      "parse development",
			input:     "development",
			expected:  EnvironmentDevelopment,
			shouldErr: false,
		},
		{
			name:      "parse staging",
			input:     "staging",
			expected:  EnvironmentStaging,
			shouldErr: false,
		},
		{
			name:      "parse production",
			input:     "production",
			expected:  EnvironmentProduction,
			shouldErr: false,
		},
		{
			name:      "parse testing",
			input:     "testing",
			expected:  EnvironmentTesting,
			shouldErr: false,
		},
		{
			name:      "parse invalid environment",
			input:     "invalid",
			expected:  Environment(""),
			shouldErr: true,
		},
		{
			name:      "parse empty string",
			input:     "",
			expected:  Environment(""),
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEnvironment(tt.input)
			if tt.shouldErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidEnvironment)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestEnvironmentValues(t *testing.T) {
	values := EnvironmentValues()

	assert.Len(t, values, 4)
	assert.Contains(t, values, EnvironmentDevelopment)
	assert.Contains(t, values, EnvironmentStaging)
	assert.Contains(t, values, EnvironmentProduction)
	assert.Contains(t, values, EnvironmentTesting)
}

func TestLoad_WithDebugLevel(t *testing.T) {
	// Save and restore original getConfigFS
	origFS := getConfigFS
	defer func() { getConfigFS = origFS }()

	// Mock filesystem with config files
	getConfigFS = func() fs.FS {
		return fstest.MapFS{
			"configs/default.toml": &fstest.MapFile{
				Data: []byte(`log-level = "debug"
port = "8080"

[database]
user = "rcv"
host = "localhost"
port = "5432"
name = "testdb"
`),
			},
		}
	}

	ctx := context.Background()
	envs := map[string]string{
		"DB_PASSWORD": "secret123",
		"APP_ENV":     "production",
		"APP_NAME":    "TestApp",
	}

	config, err := Load(ctx, envs)

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, LogLevel("debug"), config.LogLevel)
	assert.Equal(t, slog.LevelDebug, config.LogLevel.SlogLevel())
}

func TestLoad_WithDifferentDebugLevels(t *testing.T) {
	tests := []struct {
		name          string
		levelString   string
		expectedLevel slog.Level
	}{
		{
			name:          "debug level",
			levelString:   "debug",
			expectedLevel: slog.LevelDebug,
		},
		{
			name:          "info level",
			levelString:   "info",
			expectedLevel: slog.LevelInfo,
		},
		{
			name:          "warn level",
			levelString:   "warn",
			expectedLevel: slog.LevelWarn,
		},
		{
			name:          "error level",
			levelString:   "error",
			expectedLevel: slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original getConfigFS
			origFS := getConfigFS
			defer func() { getConfigFS = origFS }()

			// Mock filesystem
			getConfigFS = func() fs.FS {
				return fstest.MapFS{
					"configs/default.toml": &fstest.MapFile{
						Data: fmt.Appendf(nil, `log-level = "%s"
port = "8080"

[database]
user = "rcv"
host = "localhost"
port = "5432"
name = "testdb"
`, tt.levelString),
					},
				}
			}

			ctx := context.Background()
			envs := map[string]string{
				"DB_PASSWORD": "secret123",
				"APP_ENV":     "development",
				"APP_NAME":    "TestApp",
			}

			config, err := Load(ctx, envs)

			require.NoError(t, err)
			require.NotNil(t, config)
			assert.Equal(t, LogLevel(tt.levelString), config.LogLevel)
			assert.Equal(t, tt.expectedLevel, config.LogLevel.SlogLevel())
		})
	}
}

func TestLoad_AllEnvironments(t *testing.T) {
	tests := []struct {
		name        string
		appEnv      string
		expectedEnv Environment
	}{
		{
			name:        "development environment",
			appEnv:      "development",
			expectedEnv: EnvironmentDevelopment,
		},
		{
			name:        "staging environment",
			appEnv:      "staging",
			expectedEnv: EnvironmentStaging,
		},
		{
			name:        "production environment",
			appEnv:      "production",
			expectedEnv: EnvironmentProduction,
		},
		{
			name:        "testing environment",
			appEnv:      "testing",
			expectedEnv: EnvironmentTesting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original getConfigFS
			origFS := getConfigFS
			defer func() { getConfigFS = origFS }()

			// Mock filesystem
			getConfigFS = func() fs.FS {
				return fstest.MapFS{
					"configs/default.toml": &fstest.MapFile{
						Data: []byte(`port = "8080"

[database]
user = "rcv"
host = "localhost"
port = "5432"
name = "testdb"
`),
					},
				}
			}

			ctx := context.Background()
			envs := map[string]string{
				"DB_PASSWORD": "secret123",
				"APP_ENV":     tt.appEnv,
				"APP_NAME":    "TestApp",
			}

			config, err := Load(ctx, envs)

			require.NoError(t, err)
			require.NotNil(t, config)
			assert.Equal(t, tt.expectedEnv, config.Environment)
			assert.Equal(t, tt.appEnv, config.Environment.String())
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected string
	}{
		{
			name:     "debug level",
			level:    LogLevel("debug"),
			expected: "debug",
		},
		{
			name:     "info level",
			level:    LogLevel("info"),
			expected: "info",
		},
		{
			name:     "warn level",
			level:    LogLevel("warn"),
			expected: "warn",
		},
		{
			name:     "error level",
			level:    LogLevel("error"),
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.level.String()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLogLevel_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected bool
	}{
		{
			name:     "valid debug",
			level:    LogLevel("debug"),
			expected: true,
		},
		{
			name:     "valid info",
			level:    LogLevel("info"),
			expected: true,
		},
		{
			name:     "valid warn",
			level:    LogLevel("warn"),
			expected: true,
		},
		{
			name:     "valid error",
			level:    LogLevel("error"),
			expected: true,
		},
		{
			name:     "invalid level",
			level:    LogLevel("invalid"),
			expected: false,
		},
		{
			name:     "empty string",
			level:    LogLevel(""),
			expected: false,
		},
		{
			name:     "uppercase",
			level:    LogLevel("DEBUG"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.level.IsValid()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestLogLevel_Level(t *testing.T) {
	tests := []struct {
		name     string
		logLevel LogLevel
		expected slog.Level
	}{
		{
			name:     "debug level",
			logLevel: LogLevel("debug"),
			expected: slog.LevelDebug,
		},
		{
			name:     "info level",
			logLevel: LogLevel("info"),
			expected: slog.LevelInfo,
		},
		{
			name:     "warn level",
			logLevel: LogLevel("warn"),
			expected: slog.LevelWarn,
		},
		{
			name:     "error level",
			logLevel: LogLevel("error"),
			expected: slog.LevelError,
		},
		{
			name:     "invalid defaults to info",
			logLevel: LogLevel("invalid"),
			expected: slog.LevelInfo,
		},
		{
			name:     "empty defaults to info",
			logLevel: LogLevel(""),
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.logLevel.SlogLevel()
			assert.Equal(t, tt.expected, got)
		})
	}
}
