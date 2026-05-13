//go:build dev

package main

import "testing"

func TestBuildResetPGAdminConfigFromKeywordDSN(t *testing.T) {
	connCfg, dbName, err := buildResetPGAdminConfig("host=10.10.24.11 port=33000 user=postgres password=password dbname=metis sslmode=disable")
	if err != nil {
		t.Fatalf("build admin config: %v", err)
	}
	if dbName != "metis" {
		t.Fatalf("expected dbName metis, got %q", dbName)
	}
	if connCfg.Host != "10.10.24.11" {
		t.Fatalf("expected host 10.10.24.11, got %q", connCfg.Host)
	}
	if connCfg.Port != 33000 {
		t.Fatalf("expected port 33000, got %d", connCfg.Port)
	}
	if connCfg.Database != "template1" {
		t.Fatalf("expected admin database template1, got %q", connCfg.Database)
	}
}

func TestBuildResetPGAdminConfigFromURLDSN(t *testing.T) {
	connCfg, dbName, err := buildResetPGAdminConfig("postgres://postgres:password@10.10.24.11:33000/metis?sslmode=disable")
	if err != nil {
		t.Fatalf("build admin config: %v", err)
	}
	if dbName != "metis" {
		t.Fatalf("expected dbName metis, got %q", dbName)
	}
	if connCfg.Database != "template1" {
		t.Fatalf("expected admin database template1, got %q", connCfg.Database)
	}
}

func TestBuildResetPGAdminConfigRequiresDatabaseName(t *testing.T) {
	if _, _, err := buildResetPGAdminConfig("host=10.10.24.11 port=33000 user=postgres password=password sslmode=disable"); err == nil {
		t.Fatal("expected error when dbname is missing")
	}
}
