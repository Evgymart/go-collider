package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL     string
	ServerPort      string
	AppEnv          string
	CacheURL        string
	CachePoolSize   int
	SnowflakeNodeID int64
}

func Load() (*Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	cachePoolSize := 10
	if poolSizeStr := os.Getenv("CACHE_POOL_SIZE"); poolSizeStr != "" {
		if size, err := strconv.Atoi(poolSizeStr); err == nil && size > 0 {
			cachePoolSize = size
		}
	}

	snowflakeNodeID := int64(1)
	if nodeIDStr := os.Getenv("SNOWFLAKE_NODE_ID"); nodeIDStr != "" {
		if nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64); err == nil && nodeID >= 0 && nodeID <= 1023 {
			snowflakeNodeID = nodeID
		}
	}

	return &Config{
		DatabaseURL:     databaseURL,
		ServerPort:      serverPort,
		AppEnv:          os.Getenv("APP_ENV"),
		CacheURL:        os.Getenv("CACHE_URL"),
		CachePoolSize:   cachePoolSize,
		SnowflakeNodeID: snowflakeNodeID,
	}, nil
}
