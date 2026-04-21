package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port                          string
	TrustedProxyIps               []string
	DatabaseURL                   string
	JWTSecret                     string
	JWTIssuer                     string
	JWTAudience                   string
	BCryptCost                    int
	LoginRateLimit                int
	LoginRateLimitWindow          time.Duration
	RegisterRateLimit             int
	RegisterRateLimitWindow       time.Duration
	PasswordUpdateRateLimit       int
	PasswordUpdateRateLimitWindow time.Duration
	PermissionCacheTTL            time.Duration
	PermissionCacheMaxSize        int
}

func Load() (*Config, error) {
	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	viper.SetDefault("PORT", "8080")
	viper.SetDefault("TRUSTED_PROXY_IPS", "")
	viper.SetDefault("JWT_ISSUER", "serenity")
	viper.SetDefault("JWT_AUDIENCE", "serenity")
	viper.SetDefault("BCRYPT_COST", 12)
	viper.SetDefault("LOGIN_RATE_LIMIT", 5)
	viper.SetDefault("LOGIN_RATE_LIMIT_WINDOW", "1m")
	viper.SetDefault("REGISTER_RATE_LIMIT", 3)
	viper.SetDefault("REGISTER_RATE_LIMIT_WINDOW", "1m")
	viper.SetDefault("PASSWORD_UPDATE_RATE_LIMIT", 3)
	viper.SetDefault("PASSWORD_UPDATE_RATE_LIMIT_WINDOW", "1m")
	viper.SetDefault("PERMISSION_CACHE_TTL", "45s")
	viper.SetDefault("PERMISSION_CACHE_MAX_SIZE", "1000")

	var missing []string
	for _, key := range []string{"DATABASE_URL", "JWT_SECRET"} {
		if !viper.IsSet(key) {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	bCryptCost := viper.GetInt("BCRYPT_COST")
	if bCryptCost < 10 || bCryptCost > 14 {
		return nil, errors.New("invalid BCRYPT_COST: Must be between 10 and 14")
	}

	loginWindow, err := time.ParseDuration(viper.GetString("LOGIN_RATE_LIMIT_WINDOW"))
	if err != nil {
		return nil, fmt.Errorf("invalid LOGIN_RATE_LIMIT_WINDOW: %w", err)
	}
	registerWindow, err := time.ParseDuration(viper.GetString("REGISTER_RATE_LIMIT_WINDOW"))
	if err != nil {
		return nil, fmt.Errorf("invalid REGISTER_RATE_LIMIT_WINDOW: %w", err)
	}
	passwordUpdateWindow, err := time.ParseDuration(viper.GetString("PASSWORD_UPDATE_RATE_LIMIT_WINDOW"))
	if err != nil {
		return nil, fmt.Errorf("invalid PASSWORD_UPDATE_RATE_LIMIT_WINDOW: %w", err)
	}
	permissionCacheTTL, err := time.ParseDuration(viper.GetString("PERMISSION_CACHE_TTL"))
	if err != nil {
		return nil, fmt.Errorf("invalid PERMISSION_CACHE_TTL: %w", err)
	}

	return &Config{
		Port:                          viper.GetString("PORT"),
		TrustedProxyIps:               viper.GetStringSlice("TRUSTED_PROXY_IPS"),
		DatabaseURL:                   viper.GetString("DATABASE_URL"),
		JWTSecret:                     viper.GetString("JWT_SECRET"),
		JWTIssuer:                     viper.GetString("JWT_ISSUER"),
		JWTAudience:                   viper.GetString("JWT_AUDIENCE"),
		BCryptCost:                    bCryptCost,
		LoginRateLimit:                viper.GetInt("LOGIN_RATE_LIMIT"),
		LoginRateLimitWindow:          loginWindow,
		RegisterRateLimit:             viper.GetInt("REGISTER_RATE_LIMIT"),
		RegisterRateLimitWindow:       registerWindow,
		PasswordUpdateRateLimit:       viper.GetInt("PASSWORD_UPDATE_RATE_LIMIT"),
		PasswordUpdateRateLimitWindow: passwordUpdateWindow,
		PermissionCacheTTL:            permissionCacheTTL,
		PermissionCacheMaxSize:        viper.GetInt("PERMISSION_CACHE_MAX_SIZE"),
	}, nil
}
