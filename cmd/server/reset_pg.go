//go:build dev

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"metis/internal/config"
)

const resetPGTimeout = 30 * time.Second

func runResetPGCommand(args []string) {
	fs := flag.NewFlagSet("reset-pg", flag.ExitOnError)
	configPath := fs.String("config", "config.yml", "path to config file")
	envPath := fs.String("env", devAIConfigPath, "path to dev environment file")
	fs.Parse(args)

	if err := runResetPG(*configPath, *envPath); err != nil {
		slog.Error("reset-pg failed", "error", err)
		os.Exit(1)
	}
	slog.Info("reset-pg: all done")
}

func runResetPG(configPath, envPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		if errors.Is(err, config.ErrConfigNotFound) {
			return fmt.Errorf("%s not found", configPath)
		}
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.DBDriver != "postgres" {
		return fmt.Errorf("reset-pg only supports postgres, got %q", cfg.DBDriver)
	}

	adminCfg, dbName, err := buildResetPGAdminConfig(cfg.DBDSN)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), resetPGTimeout)
	defer cancel()

	conn, err := pgx.ConnectConfig(ctx, adminCfg)
	if err != nil {
		return fmt.Errorf("connect postgres admin database: %w", err)
	}
	defer conn.Close(context.Background())

	quotedDBName := pgx.Identifier{dbName}.Sanitize()
	if _, err := conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quotedDBName)); err != nil {
		return fmt.Errorf("drop database %s: %w", dbName, err)
	}
	if _, err := conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", quotedDBName)); err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}

	if err := runSeedDev(configPath, envPath); err != nil {
		return fmt.Errorf("seed-dev after reset: %w", err)
	}
	return nil
}

func buildResetPGAdminConfig(dsn string) (*pgx.ConnConfig, string, error) {
	connCfg, err := pgx.ParseConfig(strings.TrimSpace(dsn))
	if err != nil {
		return nil, "", fmt.Errorf("parse postgres dsn: %w", err)
	}
	dbName := strings.TrimSpace(connCfg.Database)
	if dbName == "" {
		return nil, "", fmt.Errorf("config.yml db_dsn must include dbname/database")
	}
	connCfg.Database = "template1"
	return connCfg, dbName, nil
}
