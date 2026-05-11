package engine

import (
	"testing"
	"time"
)

func TestParseSmartServiceConfigContracts(t *testing.T) {
	t.Run("empty config uses durable defaults", func(t *testing.T) {
		cfg := ParseSmartServiceConfig("")
		if cfg.ConfidenceThreshold != DefaultConfidenceThreshold {
			t.Fatalf("ConfidenceThreshold = %v, want %v", cfg.ConfidenceThreshold, DefaultConfidenceThreshold)
		}
		if cfg.DecisionTimeoutSeconds != DefaultDecisionTimeoutSeconds {
			t.Fatalf("DecisionTimeoutSeconds = %d, want %d", cfg.DecisionTimeoutSeconds, DefaultDecisionTimeoutSeconds)
		}
		if cfg.FallbackStrategy != "manual_queue" {
			t.Fatalf("FallbackStrategy = %q, want %q", cfg.FallbackStrategy, "manual_queue")
		}
	})

	t.Run("invalid and zero values degrade back to defaults", func(t *testing.T) {
		cfg := ParseSmartServiceConfig(`{"confidence_threshold":0,"decision_timeout_seconds":-12,"fallback_strategy":"manual_override"}`)
		if cfg.ConfidenceThreshold != DefaultConfidenceThreshold {
			t.Fatalf("ConfidenceThreshold = %v, want %v", cfg.ConfidenceThreshold, DefaultConfidenceThreshold)
		}
		if cfg.DecisionTimeoutSeconds != DefaultDecisionTimeoutSeconds {
			t.Fatalf("DecisionTimeoutSeconds = %d, want %d", cfg.DecisionTimeoutSeconds, DefaultDecisionTimeoutSeconds)
		}
		if cfg.FallbackStrategy != "manual_override" {
			t.Fatalf("FallbackStrategy = %q, want %q", cfg.FallbackStrategy, "manual_override")
		}
	})

	t.Run("well-formed positive values are preserved", func(t *testing.T) {
		cfg := ParseSmartServiceConfig(`{"confidence_threshold":0.91,"decision_timeout_seconds":45,"fallback_strategy":"escalate"}`)
		if cfg.ConfidenceThreshold != 0.91 {
			t.Fatalf("ConfidenceThreshold = %v, want 0.91", cfg.ConfidenceThreshold)
		}
		if cfg.DecisionTimeoutSeconds != 45 {
			t.Fatalf("DecisionTimeoutSeconds = %d, want 45", cfg.DecisionTimeoutSeconds)
		}
		if cfg.FallbackStrategy != "escalate" {
			t.Fatalf("FallbackStrategy = %q, want %q", cfg.FallbackStrategy, "escalate")
		}
	})
}

func TestResolveConvergenceTimeoutContracts(t *testing.T) {
	engine := &SmartEngine{
		configProvider: &mockConfigProvider{convergenceTimeout: 6 * time.Hour},
	}

	t.Run("future SLA resolution deadline wins over config timeout", func(t *testing.T) {
		deadline := time.Now().Add(90 * time.Minute)
		timeout := engine.resolveConvergenceTimeout(nil, &ticketModel{SLAResolutionDeadline: &deadline})
		if timeout < 89*time.Minute || timeout > 91*time.Minute {
			t.Fatalf("timeout = %s, want about 90m", timeout)
		}
	})

	t.Run("expired SLA deadline falls back to configured timeout", func(t *testing.T) {
		deadline := time.Now().Add(-30 * time.Minute)
		timeout := engine.resolveConvergenceTimeout(nil, &ticketModel{SLAResolutionDeadline: &deadline})
		if timeout != 6*time.Hour {
			t.Fatalf("timeout = %s, want %s", timeout, 6*time.Hour)
		}
	})

	t.Run("missing SLA and non-positive config fall back to hardcoded week", func(t *testing.T) {
		noConfigEngine := &SmartEngine{
			configProvider: &mockConfigProvider{convergenceTimeout: 0},
		}
		timeout := noConfigEngine.resolveConvergenceTimeout(nil, &ticketModel{})
		if timeout != 168*time.Hour {
			t.Fatalf("timeout = %s, want %s", timeout, 168*time.Hour)
		}
	})
}

func TestNormalizedDecisionModeContracts(t *testing.T) {
	if got := normalizedDecisionMode(""); got != "direct_first" {
		t.Fatalf("normalizedDecisionMode(\"\") = %q, want %q", got, "direct_first")
	}
	if got := normalizedDecisionMode("ai_only"); got != "ai_only" {
		t.Fatalf("normalizedDecisionMode(\"ai_only\") = %q, want %q", got, "ai_only")
	}
}
