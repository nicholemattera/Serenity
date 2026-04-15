package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	BCryptCost  int
}

func Load() (*Config, error) {
	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	viper.SetDefault("PORT", "8080")
	viper.SetDefault("BCRYPT_COST", 12)

	var missing []string
	for _, key := range []string{"DATABASE_URL", "JWT_SECRET"} {
		if !viper.IsSet(key) {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return &Config{
		Port:        viper.GetString("PORT"),
		DatabaseURL: viper.GetString("DATABASE_URL"),
		JWTSecret:   viper.GetString("JWT_SECRET"),
		BCryptCost:  viper.GetInt("BCRYPT_COST"),
	}, nil
}
