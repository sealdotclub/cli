package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultSealedPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"doc.pdf", "doc.pdf.sealed.pdf"},
		{"/tmp/a.PDF", "/tmp/a.PDF.sealed.pdf"},
		{"noext", "noext.sealed.pdf"},
	}
	for _, tt := range tests {
		if got := defaultSealedPath(tt.in); got != tt.want {
			t.Fatalf("defaultSealedPath(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestResolveOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		inputPath     string
		output        string
		replace       bool
		stdoutIsTTY   bool
		wantDest      string
		wantStdout    bool
		wantErrSubstr string
	}{
		{
			name:        "tty file defaults to sealed path",
			inputPath:   "doc.pdf",
			stdoutIsTTY: true,
			wantDest:    "doc.pdf.sealed.pdf",
		},
		{
			name:        "redirected file goes to stdout",
			inputPath:   "doc.pdf",
			stdoutIsTTY: false,
			wantStdout:  true,
		},
		{
			name:      "explicit output",
			inputPath: "doc.pdf",
			output:    "out.pdf",
			wantDest:  "out.pdf",
		},
		{
			name:      "replace",
			inputPath: "doc.pdf",
			replace:   true,
			wantDest:  "doc.pdf",
		},
		{
			name:       "stdin defaults to stdout",
			inputPath:  "",
			wantStdout: true,
		},
		{
			name:          "replace with stdin errors",
			replace:       true,
			wantErrSubstr: "--replace",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dest, useStdout, err := resolveOutput(tt.inputPath, tt.output, tt.replace, tt.stdoutIsTTY)
			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("error=%v want substr %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if dest != tt.wantDest || useStdout != tt.wantStdout {
				t.Fatalf("got dest=%q stdout=%v want dest=%q stdout=%v", dest, useStdout, tt.wantDest, tt.wantStdout)
			}
		})
	}
}

func TestRunVersionAndUsage(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run([]string{"-version"}, bytes.NewReader(nil), &stdout, &stderr, true, true)
	if code != 0 {
		t.Fatalf("version exit=%d stderr=%s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != version {
		t.Fatalf("version=%q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-h"}, bytes.NewReader(nil), &stdout, &stderr, true, true)
	if code != 0 {
		t.Fatalf("help exit=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage: sealclub") {
		t.Fatalf("help missing usage: %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(nil, bytes.NewReader(nil), &stdout, &stderr, true, true)
	if code != 2 {
		t.Fatalf("no input exit=%d stderr=%s", code, stderr.String())
	}
}

func TestReadInputFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, gotPath, err := readInput(path, bytes.NewReader(nil), true)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != path || string(data) != "%PDF-1.4" {
		t.Fatalf("path=%q data=%q", gotPath, data)
	}
}
