// Command sealclub seals a PDF via the seal.club API.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"charm.land/fang/v2"
	sealclub "github.com/sealdotclub/go-sdk"
	"github.com/spf13/cobra"
)

// Set by goreleaser ldflags.
var version = "dev"

func main() {
	cmd := newRoot(
		os.Stdin,
		os.Stdout,
		os.Stderr,
		isTerminal(os.Stdin),
		isTerminal(os.Stdout),
		isTerminal(os.Stderr),
	)
	err := fang.Execute(
		context.Background(),
		cmd,
		fang.WithVersion(version),
		fang.WithNotifySignal(os.Interrupt),
	)
	if err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			os.Exit(ee.code)
		}
		os.Exit(1)
	}
}

type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func usageError(format string, args ...any) error {
	return &exitError{code: 2, msg: fmt.Sprintf(format, args...)}
}

func runtimeError(err error) error {
	if err == nil {
		return nil
	}
	return &exitError{code: 1, msg: err.Error()}
}

func newRoot(stdin io.Reader, stdout, stderr io.Writer, stdinIsTTY, stdoutIsTTY, stderrIsTTY bool) *cobra.Command {
	var (
		output  string
		replace bool
		quiet   bool
	)

	cmd := &cobra.Command{
		Use:   "sealclub [input.pdf|-]",
		Short: "Seal a PDF with seal.club",
		Long: `Seal a PDF with the seal.club API.

Pass a file path, or omit it / use "-" to read PDF bytes from stdin.
File input on a terminal writes <name>.sealed.pdf by default.
Redirected stdout (or stdin input) writes the sealed PDF to stdout.

Requires SEAL_API_KEY. Optional SEAL_API_BASE_URL (default https://api.seal.club).`,
		Example: `  # file → file
  sealclub doc.pdf
  sealclub doc.pdf -o sealed.pdf

  # pipes
  cat doc.pdf | sealclub > sealed.pdf
  sealclub doc.pdf > sealed.pdf

  # in-place / quiet
  sealclub doc.pdf --replace
  sealclub doc.pdf --quiet`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			inputArg := ""
			if len(args) == 1 {
				inputArg = args[0]
			}

			// No args on a terminal → help (not an error).
			if inputArg == "" && stdinIsTTY {
				return cmd.Help()
			}

			if output != "" && replace {
				return usageError("use either --output or --replace, not both")
			}

			status := newStatusWriter(stderr, quiet, stderrIsTTY)

			pdf, inputPath, err := readInput(inputArg, stdin, stdinIsTTY)
			if err != nil {
				return usageError("%s", err.Error())
			}

			dest, useStdout, err := resolveOutput(inputPath, output, replace, stdoutIsTTY)
			if err != nil {
				return usageError("%s", err.Error())
			}

			client, err := sealclub.FromEnv()
			if err != nil {
				return runtimeError(err)
			}

			stop := status.start("Sealing")
			sealed, err := client.Seal(pdf)
			stop()
			if err != nil {
				return runtimeError(err)
			}

			if useStdout {
				if _, err := stdout.Write(sealed); err != nil {
					return runtimeError(err)
				}
				return nil
			}

			if err := os.WriteFile(dest, sealed, 0o644); err != nil {
				return runtimeError(err)
			}
			status.success(dest)
			return nil
		},
	}

	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.Flags().StringVarP(&output, "output", "o", "", "Write sealed PDF to this path")
	cmd.Flags().BoolVar(&replace, "replace", false, "Replace the input file with the sealed PDF")
	cmd.Flags().BoolVar(&replace, "in-place", false, "Alias for --replace")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress progress spinner and status output")
	_ = cmd.Flags().MarkHidden("in-place")

	return cmd
}

// run executes the command for tests (without Fang styling).
func run(args []string, stdin io.Reader, stdout, stderr io.Writer, stdinIsTTY, stdoutIsTTY bool) int {
	cmd := newRoot(stdin, stdout, stderr, stdinIsTTY, stdoutIsTTY, false)
	cmd.Version = version
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			fmt.Fprintln(stderr, "error:", ee.msg)
			return ee.code
		}
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func readInput(inputArg string, stdin io.Reader, stdinIsTTY bool) ([]byte, string, error) {
	if inputArg == "" || inputArg == "-" {
		if inputArg == "" && stdinIsTTY {
			return nil, "", fmt.Errorf("no input provided; pass a PDF path or pipe PDF bytes into sealclub")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, "", err
		}
		if len(data) == 0 {
			return nil, "", fmt.Errorf("no input provided on stdin")
		}
		return data, "", nil
	}

	info, err := os.Stat(inputArg)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", fmt.Errorf("file not found: %s", inputArg)
		}
		return nil, "", err
	}
	if info.IsDir() {
		return nil, "", fmt.Errorf("not a file: %s", inputArg)
	}

	data, err := os.ReadFile(inputArg)
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("PDF body is empty")
	}
	return data, inputArg, nil
}

func resolveOutput(inputPath, output string, replace, stdoutIsTTY bool) (dest string, useStdout bool, err error) {
	if replace {
		if inputPath == "" {
			return "", false, fmt.Errorf("--replace can only be used with a file input, not stdin")
		}
		return inputPath, false, nil
	}
	if output != "" {
		return output, false, nil
	}
	if inputPath == "" {
		return "", true, nil
	}
	if !stdoutIsTTY {
		return "", true, nil
	}
	return defaultSealedPath(inputPath), false, nil
}

func defaultSealedPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	if base == "" {
		base = inputPath
	}
	return base + ".sealed.pdf"
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
