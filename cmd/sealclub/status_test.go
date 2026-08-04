package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestStatusWriterQuiet(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	s := newStatusWriter(&buf, true, true)
	stop := s.start("Sealing")
	time.Sleep(20 * time.Millisecond)
	stop()
	s.success("out.pdf")
	if buf.Len() != 0 {
		t.Fatalf("quiet wrote %q", buf.String())
	}
}

func TestStatusWriterPlainSuccess(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	s := newStatusWriter(&buf, false, false)
	stop := s.start("Sealing")
	stop()
	s.success("out.pdf")
	if got := strings.TrimSpace(buf.String()); got != "out.pdf" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestStatusWriterFancySuccess(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	s := newStatusWriter(&buf, false, true)
	s.success("out.pdf")
	out := buf.String()
	if !strings.Contains(out, "out.pdf") || !strings.Contains(out, "✓") {
		t.Fatalf("fancy success=%q", out)
	}
}

func TestHelpListsQuiet(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, bytes.NewReader(nil), &stdout, &stderr, true, true)
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	help := stdout.String() + stderr.String()
	if !strings.Contains(help, "--quiet") {
		t.Fatalf("help missing --quiet: %q", help)
	}
}
