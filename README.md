# Config

[![GoDoc](https://pkg.go.dev/badge/github.com/kaduartur/config)](https://pkg.go.dev/github.com/kaduartur/config)
[![CodeFactor](https://www.codefactor.io/repository/github/kaduartur/config/badge)](https://www.codefactor.io/repository/github/kaduartur/config)
[![codecov](https://codecov.io/gh/kaduartur/config/graph/badge.svg?token=DPUTJ35TOB)](https://codecov.io/gh/kaduartur/config)

Package `config` provides convenient access methods for configuration management stored as JSON or YAML with support for nested values via dotted path notation.

## Features

### Configuration Parsing
- **JSON and YAML Support**: Parse configuration from strings, byte slices, or files
- **Dotted Path Access**: Navigate nested configuration using simple paths like `"database.host"` or `"servers.0.port"`
- **Type-Safe Getters**: Retrieve values as specific types (Bool, Int, Float64, String, List, Map)
- **Safe Getters**: Use `U*` methods (UBool, UInt, etc.) that return default values instead of errors

### Dynamic Configuration
- **Set Values**: Modify configuration values at runtime using `Set(path, value)`
- **Copy**: Create deep copies of entire config or specific sub-paths
- **Extend**: Merge configurations with intelligent array handling

### External Sources
- **Environment Variables**: Auto-populate config from environment variables with `Env()` or `EnvPrefix()`
- **Command-Line Flags**: Parse command-line arguments with `Flag()` or `Args()`
- **Error Handling**: Access parsing errors with `Error()` method

## Installation

```bash
go get github.com/kaduartur/config
```

## Quick Start

### Parsing Configuration

```go
import "github.com/kaduartur/config"

// From YAML file
cfg, err := config.ParseYamlFile("config.yaml")

// From JSON file
cfg, err := config.ParseJsonFile("config.json")

// From string
cfg, err := config.ParseYaml(`
server:
  host: localhost
  port: 8080
database:
  hosts:
    - primary.db.com
    - replica.db.com
`)

// Using Must helper for initialization
var cfg = config.Must(config.ParseYaml(yamlString))
```

### Accessing Values

```go
// Get nested values with dotted paths
host, err := cfg.String("server.host")        // "localhost"
port, err := cfg.Int("server.port")           // 8080
dbHost, err := cfg.String("database.hosts.0") // "primary.db.com"

// Use safe getters with default values
timeout := cfg.UInt("server.timeout", 30)     // returns 30 if not found
debug := cfg.UBool("server.debug", false)     // returns false if not found
name := cfg.UString("app.name", "myapp")      // returns "myapp" if not found

// Get complex types
hosts, err := cfg.List("database.hosts")      // []any
settings, err := cfg.Map("server")            // map[string]any

// Navigate to sub-configurations
serverCfg, err := cfg.Get("server")
port, err := serverCfg.Int("port")
```

### Modifying Configuration

```go
// Set values dynamically
err := cfg.Set("server.port", 9000)
err := cfg.Set("cache.enabled", true)
err := cfg.Set("servers.0.name", "primary")

// Create deep copies
clone, err := cfg.Copy()
serverCopy, err := cfg.Copy("server")

// Extend configuration (merges arrays intelligently)
baseCfg, _ := config.ParseYaml(`
defaults:
  timeout: 30
  retries: 3
`)

appCfg, _ := config.ParseYaml(`
defaults:
  retries: 5
app:
  name: myapp
`)

merged, err := baseCfg.Extend(appCfg)
// Result contains all keys from both configs
// Primitive values are overridden, arrays are merged
```

### Environment Variables

```go
// Given config with keys: server.host, server.port, database.name
// Automatically reads from environment variables:
// SERVER_HOST, SERVER_PORT, DATABASE_NAME
cfg.Env()

// With custom prefix (reads APP_SERVER_HOST, APP_SERVER_PORT, etc.)
cfg.EnvPrefix("APP")
```

### Command-Line Arguments

```go
// Using standard flag package
cfg.Flag()
// Run with: ./myapp -server-host=0.0.0.0 -server-port=9000

// Using custom arguments
cfg.Args("myapp", "-server-host=0.0.0.0", "-server-port=9000")

// Check for parsing errors
if err := cfg.Error(); err != nil {
    log.Fatal(err)
}
```

## API Reference

### Parsing Functions

| Function | Description |
|----------|-------------|
| `ParseYaml(string) (*Config, error)` | Parse YAML from string |
| `ParseYamlBytes([]byte) (*Config, error)` | Parse YAML from byte slice |
| `ParseYamlFile(string) (*Config, error)` | Parse YAML from file |
| `ParseJson(string) (*Config, error)` | Parse JSON from string |
| `ParseJsonFile(string) (*Config, error)` | Parse JSON from file |
| `Must(*Config, error) *Config` | Helper that panics on error (for initialization) |

### Getter Methods

| Method | Return Type | Description |
|--------|-------------|-------------|
| `Get(path) (*Config, error)` | `*Config` | Get nested config |
| `Bool(path) (bool, error)` | `bool` | Get boolean value |
| `Int(path) (int, error)` | `int` | Get integer value |
| `Float64(path) (float64, error)` | `float64` | Get float value |
| `String(path) (string, error)` | `string` | Get string value |
| `List(path) ([]any, error)` | `[]any` | Get array value |
| `Map(path) (map[string]any, error)` | `map[string]any` | Get map value |

### Safe Getter Methods (with defaults)

| Method | Return Type | Description |
|--------|-------------|-------------|
| `UBool(path, ...bool) bool` | `bool` | Returns value or default or false |
| `UInt(path, ...int) int` | `int` | Returns value or default or 0 |
| `UFloat64(path, ...float64) float64` | `float64` | Returns value or default or 0.0 |
| `UString(path, ...string) string` | `string` | Returns value or default or "" |
| `UList(path, ...[]any) []any` | `[]any` | Returns value or default or empty slice |
| `UMap(path, ...map[string]any) map[string]any` | `map[string]any` | Returns value or default or empty map |

### Modification Methods

| Method | Description |
|--------|-------------|
| `Set(path, value) error` | Set value at dotted path |
| `Copy(...path) (*Config, error)` | Create deep copy of config or sub-path |
| `Extend(*Config) (*Config, error)` | Merge another config (intelligently merges arrays) |

### External Source Methods

| Method | Description |
|--------|-------------|
| `Env() *Config` | Load values from environment variables |
| `EnvPrefix(prefix) *Config` | Load values from environment variables with prefix |
| `Flag() *Config` | Parse command-line flags using standard flag package |
| `Args(...string) *Config` | Parse command-line arguments |
| `Error() error` | Get last parsing error from Args() |

### Rendering Methods

| Function | Description |
|----------|-------------|
| `RenderYaml(any) (string, error)` | Convert config to YAML string |
| `RenderJson(any) (string, error)` | Convert config to JSON string |

## Path Notation

The package uses dotted path notation to access nested values:

```go
cfg := config.Must(config.ParseYaml(`
database:
  primary:
    host: db1.example.com
    port: 5432
  replicas:
    - host: db2.example.com
      port: 5432
    - host: db3.example.com
      port: 5433
`))

// Access nested map values
host, _ := cfg.String("database.primary.host") // "db1.example.com"

// Access array elements by index
replica1, _ := cfg.String("database.replicas.0.host") // "db2.example.com"
port, _ := cfg.Int("database.replicas.1.port")        // 5433

// Use bracket notation for array access
replica2, _ := cfg.String("database.replicas[1].host") // "db3.example.com"
```

## Advanced Usage

### Configuration Layers

Combine base configurations with environment-specific overrides:

```go
// Load base configuration
base, _ := config.ParseYamlFile("config.base.yaml")

// Load environment-specific config
envCfg, _ := config.ParseYamlFile("config.production.yaml")

// Merge configurations
final, _ := base.Extend(envCfg)

// Override with environment variables
final.EnvPrefix("APP")

// Override with command-line flags
final.Flag()
```

### Type Conversions

The package automatically handles type conversions where possible:

```go
// String to bool
cfg.Set("enabled", "true")
enabled, _ := cfg.Bool("enabled") // true

// String to number
cfg.Set("port", "8080")
port, _ := cfg.Int("port") // 8080

// Number to string
cfg.Set("count", 42)
str, _ := cfg.String("count") // "42"
```

## License

This package is based on the original work by [moraes/config](https://github.com/moraes/config) and extended by [olebedev/config](https://github.com/olebedev/config).
