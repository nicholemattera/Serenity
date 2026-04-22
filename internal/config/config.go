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
	ReadHeaderTimeout             time.Duration
	ReadTimeout                   time.Duration
	WriteTimeout                  time.Duration
	IdleTimeout                   time.Duration
	MaxBodyBytes                  int64
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
	viper.SetDefault("READ_HEADER_TIMEOUT", "5s")
	viper.SetDefault("READ_TIMEOUT", "30s")
	viper.SetDefault("WRITE_TIMEOUT", "30s")
	viper.SetDefault("IDLE_TIMEOUT", "120s")
	viper.SetDefault("MAX_BODY_BYTES", 1048576)
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

	readHeaderTimeout, err := time.ParseDuration(viper.GetString("READ_HEADER_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid READ_HEADER_TIMEOUT: %w", err)
	}
	readTimeout, err := time.ParseDuration(viper.GetString("READ_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid READ_TIMEOUT: %w", err)
	}
	writeTimeout, err := time.ParseDuration(viper.GetString("WRITE_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid WRITE_TIMEOUT: %w", err)
	}
	idleTimeout, err := time.ParseDuration(viper.GetString("IDLE_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid IDLE_TIMEOUT: %w", err)
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
		ReadHeaderTimeout:             readHeaderTimeout,
		ReadTimeout:                   readTimeout,
		WriteTimeout:                  writeTimeout,
		IdleTimeout:                   idleTimeout,
		MaxBodyBytes:                  viper.GetInt64("MAX_BODY_BYTES"),
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
