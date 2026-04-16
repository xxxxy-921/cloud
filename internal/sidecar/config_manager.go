package sidecar

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type ConfigFileSpec struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type ConfigManager struct {
	client    *Client
	pm        *ProcessManager
	outputDir string
	hashes    map[string]string // "processName/filename" → last known hash
	mu        sync.Mutex
}

func NewConfigManager(client *Client, pm *ProcessManager, outputDir string) *ConfigManager {
	return &ConfigManager{
		client:    client,
		pm:        pm,
		outputDir: outputDir,
		hashes:    make(map[string]string),
	}
}

// hashKey returns the hash map key for a process + filename pair.
func hashKey(processName, filename string) string {
	if filename == "" {
		return processName
	}
	return processName + "/" + filename
}

// SyncConfig downloads all config files for a process, compares hashes, writes if changed, and triggers reload.
func (cm *ConfigManager) SyncConfig(defID uint, processName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Get config file list from the managed process
	configFiles := cm.getConfigFileSpecs(defID)

	if len(configFiles) == 0 {
		// Backward compatible: single unnamed config file
		updated, err := cm.syncSingleFile(defID, processName, "")
		if err != nil {
			return err
		}
		if updated {
			if err := cm.pm.Reload(defID); err != nil {
				slog.Warn("reload after config update failed", "process", processName, "error", err)
			}
		}
		return nil
	}

	changed := false
	for _, cf := range configFiles {
		updated, err := cm.syncSingleFile(defID, processName, cf.Filename)
		if err != nil {
			return err
		}
		if updated {
			changed = true
		}
	}

	if changed {
		if err := cm.pm.Reload(defID); err != nil {
			slog.Warn("reload after config update failed", "process", processName, "error", err)
		}
	}
	return nil
}

// syncSingleFile downloads and writes a single config file. Returns true if the file was updated.
func (cm *ConfigManager) syncSingleFile(defID uint, processName string, filename string) (bool, error) {
	resp, err := cm.client.DownloadConfig(processName, filename)
	if err != nil {
		return false, fmt.Errorf("download config for %s/%s: %w", processName, filename, err)
	}

	serverHash := resp.Hash
	if serverHash == "" {
		serverHash = fmt.Sprintf("%x", sha256.Sum256([]byte(resp.Content)))
	}

	key := hashKey(processName, filename)
	if cm.hashes[key] == serverHash {
		slog.Debug("config unchanged", "process", processName, "file", filename)
		return false, nil
	}

	// Write config file
	dir := filepath.Join(cm.outputDir, processName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}

	outFilename := filename
	if outFilename == "" {
		outFilename = "config"
	}
	configPath := filepath.Join(dir, outFilename)
	if err := os.WriteFile(configPath, []byte(resp.Content), 0644); err != nil {
		return false, fmt.Errorf("write config file: %w", err)
	}

	cm.hashes[key] = serverHash
	cm.pm.SetConfigVersion(defID, serverHash)

	slog.Info("config updated", "process", processName, "file", outFilename, "hash", serverHash[:12])
	return true, nil
}

// getConfigFileSpecs retrieves the config file list from the process definition in memory.
func (cm *ConfigManager) getConfigFileSpecs(defID uint) []ConfigFileSpec {
	cm.pm.mu.RLock()
	mp, ok := cm.pm.processes[defID]
	cm.pm.mu.RUnlock()
	if !ok {
		return nil
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	if len(mp.Def.ConfigFiles) == 0 {
		return nil
	}

	var specs []ConfigFileSpec
	_ = json.Unmarshal(mp.Def.ConfigFiles, &specs)
	return specs
}

// HandleConfigUpdate processes a config.update command.
func (cm *ConfigManager) HandleConfigUpdate(defID uint, processName string, client *Client, cmdID uint) {
	err := cm.SyncConfig(defID, processName)
	success := err == nil
	result := ""
	if err != nil {
		result = err.Error()
		slog.Error("config update failed", "process", processName, "error", err)
	}

	if ackErr := client.AckCommand(cmdID, success, result); ackErr != nil {
		slog.Error("failed to ack config.update command", "id", cmdID, "error", ackErr)
	}
}
