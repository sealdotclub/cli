package main

import (
	"fmt"
	"io"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

var (
	spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	spinnerStyle = lipgloss.NewStyle().Foreground(charmtone.Cheeky).Bold(true)
	labelStyle   = lipgloss.NewStyle().Foreground(charmtone.Ash)
	successStyle = lipgloss.NewStyle().Foreground(charmtone.Guac).Bold(true)
	pathStyle    = lipgloss.NewStyle().Foreground(charmtone.Malibu)
)

// statusWriter prints progress to stderr.
// Spinner/styling only when stderr is a TTY and --quiet is off.
type statusWriter struct {
	w     io.Writer
	quiet bool
	fancy bool
	mu    sync.Mutex
}

func newStatusWriter(w io.Writer, quiet, isTTY bool) *statusWriter {
	return &statusWriter{
		w:     w,
		quiet: quiet,
		fancy: !quiet && isTTY,
	}
}

// start begins an animated spinner. Call the returned stop func when done.
func (s *statusWriter) start(label string) (stop func()) {
	if s == nil || !s.fancy {
		return func() {}
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(70 * time.Millisecond)
		defer ticker.Stop()

		s.mu.Lock()
		_, _ = fmt.Fprint(s.w, "\033[?25l")
		s.mu.Unlock()

		i := 0
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				frame := spinnerStyle.Render(spinnerFrames[i%len(spinnerFrames)])
				text := labelStyle.Render(label + "…")
				s.mu.Lock()
				_, _ = fmt.Fprintf(s.w, "\r\033[K%s %s", frame, text)
				s.mu.Unlock()
				i++
			}
		}
	}()

	return func() {
		close(done)
		wg.Wait()
		s.mu.Lock()
		_, _ = fmt.Fprint(s.w, "\r\033[K\033[?25h")
		s.mu.Unlock()
	}
}

func (s *statusWriter) success(path string) {
	if s == nil || s.quiet || path == "" {
		return
	}
	if s.fancy {
		mark := successStyle.Render("✓")
		body := pathStyle.Render(path)
		s.mu.Lock()
		_, _ = fmt.Fprintf(s.w, "%s %s\n", mark, body)
		s.mu.Unlock()
		return
	}
	_, _ = fmt.Fprintln(s.w, path)
}
