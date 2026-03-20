package cli

import (
	"testing"
	"time"

	"github.com/cyperx84/clawrus/internal/types"
)

// ── applyTemplate ─────────────────────────────────────────────────────────────

func TestApplyTemplate_AllPlaceholders(t *testing.T) {
	now := time.Now()
	result := applyTemplate("hello {{name}} in {{group}} via {{preset}} on {{date}} at {{time}}", "MyThread", "mygroup", "mypreset")
	expected_date := now.Format("2006-01-02")
	expected_time := now.Format("15:04")

	if !contains(result, "MyThread") {
		t.Errorf("expected {{name}} → MyThread, got: %s", result)
	}
	if !contains(result, "mygroup") {
		t.Errorf("expected {{group}} → mygroup, got: %s", result)
	}
	if !contains(result, "mypreset") {
		t.Errorf("expected {{preset}} → mypreset, got: %s", result)
	}
	if !contains(result, expected_date) {
		t.Errorf("expected {{date}} → %s, got: %s", expected_date, result)
	}
	if !contains(result, expected_time) {
		t.Errorf("expected {{time}} → %s, got: %s", expected_time, result)
	}
}

func TestApplyTemplate_NoPlaceholders(t *testing.T) {
	msg := "just a plain message"
	result := applyTemplate(msg, "name", "group", "preset")
	if result != msg {
		t.Errorf("expected unchanged message, got: %s", result)
	}
}

func TestApplyTemplate_EmptyPreset(t *testing.T) {
	result := applyTemplate("from {{preset}}", "n", "g", "")
	if result != "from " {
		t.Errorf("expected 'from ', got: %s", result)
	}
}

func TestApplyTemplate_EmptyMessage(t *testing.T) {
	result := applyTemplate("", "n", "g", "p")
	if result != "" {
		t.Errorf("expected empty string, got: %s", result)
	}
}

// ── resolveModel ──────────────────────────────────────────────────────────────

func TestResolveModel_FlagWins(t *testing.T) {
	got := resolveModel("group-model", "thread-model", "flag-model")
	if got != "flag-model" {
		t.Errorf("expected flag-model, got %s", got)
	}
}

func TestResolveModel_ThreadOverridesGroup(t *testing.T) {
	got := resolveModel("group-model", "thread-model", "")
	if got != "thread-model" {
		t.Errorf("expected thread-model, got %s", got)
	}
}

func TestResolveModel_GroupFallback(t *testing.T) {
	got := resolveModel("group-model", "", "")
	if got != "group-model" {
		t.Errorf("expected group-model, got %s", got)
	}
}

func TestResolveModel_AllEmpty(t *testing.T) {
	got := resolveModel("", "", "")
	if got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}

// ── resolveThinking ───────────────────────────────────────────────────────────

func TestResolveThinking_FlagWins(t *testing.T) {
	got := resolveThinking("low", "medium", "high")
	if got != "high" {
		t.Errorf("expected high, got %s", got)
	}
}

func TestResolveThinking_ThreadOverridesGroup(t *testing.T) {
	got := resolveThinking("low", "medium", "")
	if got != "medium" {
		t.Errorf("expected medium, got %s", got)
	}
}

func TestResolveThinking_GroupFallback(t *testing.T) {
	got := resolveThinking("low", "", "")
	if got != "low" {
		t.Errorf("expected low, got %s", got)
	}
}

// ── resolveTimeout ────────────────────────────────────────────────────────────

func TestResolveTimeout_FlagWins(t *testing.T) {
	g := intPtr(60)
	th := intPtr(120)
	got := resolveTimeout(g, th, 30)
	if got != 30*time.Second {
		t.Errorf("expected 30s, got %v", got)
	}
}

func TestResolveTimeout_ThreadOverridesGroup(t *testing.T) {
	g := intPtr(60)
	th := intPtr(120)
	got := resolveTimeout(g, th, 0)
	if got != 120*time.Second {
		t.Errorf("expected 120s, got %v", got)
	}
}

func TestResolveTimeout_GroupFallback(t *testing.T) {
	g := intPtr(60)
	got := resolveTimeout(g, nil, 0)
	if got != 60*time.Second {
		t.Errorf("expected 60s, got %v", got)
	}
}

func TestResolveTimeout_Default(t *testing.T) {
	got := resolveTimeout(nil, nil, 0)
	if got != 300*time.Second {
		t.Errorf("expected 300s default, got %v", got)
	}
}

// ── resolvePreset ─────────────────────────────────────────────────────────────

func TestResolvePreset_NotFound(t *testing.T) {
	cfg := &types.GroupConfig{Groups: map[string]types.Group{}}
	_, _, err := resolvePreset("missing", cfg)
	if err == nil {
		t.Error("expected error for missing preset, got nil")
	}
}

func TestResolvePreset_AllExpands(t *testing.T) {
	cfg := &types.GroupConfig{
		Groups: map[string]types.Group{
			"backend": {Threads: []types.Thread{{ID: "1", Name: "A"}}},
			"frontend": {Threads: []types.Thread{{ID: "2", Name: "B"}}},
		},
	}
	// @all should include threads from all groups
	g, name, err := resolvePreset("all", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name == "" {
		t.Error("expected non-empty group name")
	}
	if len(g.Threads) != 2 {
		t.Errorf("expected 2 threads from @all, got %d", len(g.Threads))
	}
}

func TestResolvePreset_DeduplicatesThreads(t *testing.T) {
	// Two groups sharing the same thread ID — should deduplicate
	cfg := &types.GroupConfig{
		Groups: map[string]types.Group{
			"g1": {Threads: []types.Thread{{ID: "shared", Name: "T"}}},
			"g2": {Threads: []types.Thread{{ID: "shared", Name: "T"}}},
		},
	}
	g, _, err := resolvePreset("all", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Threads) != 1 {
		t.Errorf("expected deduplication to 1 thread, got %d", len(g.Threads))
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub)))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func intPtr(v int) *int {
	return &v
}
