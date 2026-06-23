package detector

import (
	"testing"
	"time"

	"github.com/skhanal5/clips/internal/chat"
	"github.com/skhanal5/clips/internal/config"
)

func TestEvaluateTriggersOnSpike(t *testing.T) {
	cfg := config.Thresholds{
		EvaluationSeconds: 10,
		BaselineSeconds:   300,
		TriggerRatio:      4,
		CooldownSeconds:   300,
	}
	d := New(cfg)
	now := time.Now()

	for i := 0; i < 200; i++ {
		d.Feed(chat.Message{Channel: "test", Timestamp: now.Add(-time.Duration(300-i) * time.Second)})
	}

	for i := 0; i < 40; i++ {
		d.Feed(chat.Message{Channel: "test", Timestamp: now.Add(-time.Duration(9) * time.Second)})
	}

	d.evaluate()

	select {
	case tr := <-d.Triggers():
		if tr.Ratio < cfg.TriggerRatio {
			t.Errorf("expected ratio >= 4, got %f", tr.Ratio)
		}
	default:
		t.Error("expected trigger, got none")
	}
}

func TestEvaluateNoTriggerOnSteady(t *testing.T) {
	cfg := config.Thresholds{
		EvaluationSeconds: 10,
		BaselineSeconds:   300,
		TriggerRatio:      4,
		CooldownSeconds:   300,
	}
	d := New(cfg)
	now := time.Now()

	for i := 0; i < 60; i++ {
		d.Feed(chat.Message{Channel: "test", Timestamp: now.Add(-time.Duration(300-i*5) * time.Second)})
	}

	d.evaluate()

	select {
	case <-d.Triggers():
		t.Error("unexpected trigger on steady activity")
	default:
	}
}
