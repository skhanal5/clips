package edit

import (
	"strings"
	"testing"
)

func TestSplitTextIntoLines(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxWidth int
		want     []string
	}{
		{
			name:     "short text stays one line",
			text:     "hello",
			maxWidth: 20,
			want:     []string{"HELLO"},
		},
		{
			name:     "long text wraps",
			text:     "hello world this is a test",
			maxWidth: 10,
			want:     []string{"HELLO", "WORLD THIS", "IS A TEST"},
		},
		{
			name:     "uppercased",
			text:     "Hello World",
			maxWidth: 20,
			want:     []string{"HELLO WORLD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitTextIntoLines(tt.text, tt.maxWidth)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d lines, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildArgsBlurred(t *testing.T) {
	args := buildArgs("in.mp4", "out.mp4", Config{Background: "blurred"})

	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}

	if args[0] != "-i" || args[1] != "in.mp4" {
		t.Errorf("expected input -i in.mp4, got %s %s", args[0], args[1])
	}

	if args[len(args)-1] != "out.mp4" {
		t.Errorf("expected output file at end, got %s", args[len(args)-1])
	}

	hasFilter := false
	for i, a := range args {
		if a == "-filter_complex" && i+1 < len(args) {
			hasFilter = true
			fc := args[i+1]
			if !strings.Contains(fc, "boxblur") {
				t.Error("filter_complex missing boxblur for blurred background")
			}
			if !strings.Contains(fc, "overlay") {
				t.Error("filter_complex missing overlay")
			}
		}
	}
	if !hasFilter {
		t.Error("args missing -filter_complex")
	}
}

func TestBuildArgsTitleQuoteEscaping(t *testing.T) {
	cfg := Config{Background: "black", Title: "it's a test"}
	args := buildArgs("in.mp4", "out.mp4", cfg)

	hasFilter := false
	for i, a := range args {
		if a == "-filter_complex" && i+1 < len(args) {
			hasFilter = true
			fc := args[i+1]
			if !strings.Contains(fc, "text='IT'\\''S A TEST'") {
				t.Errorf("title not properly escaped in filter_complex:\n%s", fc)
			}
		}
	}
	if !hasFilter {
		t.Error("args missing -filter_complex")
	}
}
