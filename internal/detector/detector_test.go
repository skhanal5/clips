package detector

import (
	"testing"

	"github.com/skhanal5/clippy-agent/internal/config"
)

func TestComputeScoreClampsToMax(t *testing.T) {
	d := New(config.Thresholds{
		MessagesPerSecond: 10,
		UniqueUsers:       10,
		EmotesPerWindow:   20,
		CooldownSeconds:   300,
		EvaluationWindow:  10,
	})

	// All factors above threshold — each clamped to 1.0 => score = 1.0
	score := d.computeScore(100, 100, 100)
	if score != 1.0 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestComputeScoreZeroInput(t *testing.T) {
	d := New(config.Thresholds{
		MessagesPerSecond: 10,
		UniqueUsers:       10,
		EmotesPerWindow:   20,
		CooldownSeconds:   300,
		EvaluationWindow:  10,
	})

	score := d.computeScore(0, 0, 0)
	if score != 0.0 {
		t.Errorf("expected 0.0, got %f", score)
	}
}

func TestComputeScoreHalfThreshold(t *testing.T) {
	d := New(config.Thresholds{
		MessagesPerSecond: 10,
		UniqueUsers:       10,
		EmotesPerWindow:   20,
		CooldownSeconds:   300,
		EvaluationWindow:  10,
	})

	// Each factor at 50% of threshold => (0.5 + 0.5 + 0.5) / 3 = 0.5
	score := d.computeScore(5, 5, 10)
	if score != 0.5 {
		t.Errorf("expected 0.5, got %f", score)
	}
}
