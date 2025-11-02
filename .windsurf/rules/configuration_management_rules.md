---
trigger: model_decision
description: when working on creating/updating/handling application configuration
---

## ⚙️ Configuration Management

**Library:** [Viper](https://github.com/spf13/viper)

**Configuration File:** `config.yml`

### 1. Configuration Struct

Define a struct that represents your application's configuration in `internal/platform/config/config.go`.

```go
package config

type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    JWT      JWTConfig      `mapstructure:"jwt"`
}

type ServerConfig struct {
    Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
    Host     string `mapstructure:"host"`
    Port     string `mapstructure:"port"`
    User     string `mapstructure:"user"`
    Password string `mapstructure:"password"`
    DBName   string `mapstructure:"dbname"`
}

type JWTConfig struct {
    Secret string `mapstructure:"secret"`
}
```

### 2. Loading Configuration

Use Viper to load the configuration from a file and environment variables.

```go
package config

import (
    "github.com/spf13/viper"
)

func LoadConfig() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yml")
    viper.AddConfigPath(".")
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }

    return &config, nil
}
```

### 3. Configuration File Example (`config.yml`)

```yaml
server:
  port: "8080"

database:
  host: "localhost"
  port: "5432"
  user: "user"
  password: "password"
  dbname: "mma_db"

jwt:
  secret: "your-secret-key"
```

### Configuration Rules:

- ✅ Use Viper for configuration management.
- ✅ Define a configuration struct in `internal/platform/config/config.go`.
- ✅ Load configuration from a `config.yml` file and environment variables.
- ✅ Use `mapstructure` tags to map configuration keys to struct fields.
- ✅ Do not commit sensitive information to version control. Use environment variables for secrets.
