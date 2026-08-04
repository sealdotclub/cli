// Command sealclub seals a PDF via the seal.club API.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	sealclub "github.com/sealdotclub/go-sdk"
)

// Set by goreleaser ldflags.
var version = "dev"

func main() {
	os.Exit(run(
		os.Args[1:],
		os.Stdin,
		os.Stdout,
		os.Stderr,
		isTerminal(os.Stdin),
		isTerminal(os.Stdout),
	))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, stdinIsTTY, stdoutIsTTY bool) int {
	fs := flag.NewFlagSet("sealclub", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		output  string
		replace bool
		showVer bool
	)
	fs.StringVar(&output, "o", "", "output path")
	fs.StringVar(&output, "output", "", "output path")
	fs.BoolVar(&replace, "replace", false, "replace the input file with the sealed PDF")
	fs.BoolVar(&replace, "in-place", false, "alias for --replace")
	fs.BoolVar(&showVer, "version", false, "print version and exit")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: sealclub [options] [input.pdf|-]\n\n")
		fmt.Fprintf(stderr, "Seal a PDF with seal.club.\n\n")
		fmt.Fprintf(stderr, "If input is omitted or '-', read PDF bytes from stdin.\n")
		fmt.Fprintf(stderr, "With a file input and a terminal stdout, write <input>.sealed.pdf\n")
		fmt.Fprintf(stderr, "unless -o/--output or --replace is set. Redirected stdout receives\n")
		fmt.Fprintf(stderr, "the sealed PDF. Stdin input defaults to stdout.\n\n")
		fmt.Fprintf(stderr, "Environment:\n")
		fmt.Fprintf(stderr, "  SEAL_API_KEY         API key (required)\n")
		fmt.Fprintf(stderr, "  SEAL_API_BASE_URL    API base URL (default https://api.seal.club)\n\n")
		fmt.Fprintf(stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if showVer {
		fmt.Fprintln(stdout, version)
		return 0
	}

	positional := fs.Args()
	if len(positional) > 1 {
		fmt.Fprintf(stderr, "error: unexpected arguments: %s\n", strings.Join(positional[1:], " "))
		fs.Usage()
		return 2
	}

	inputArg := ""
	if len(positional) == 1 {
		inputArg = positional[0]
	}

	if output != "" && replace {
		fmt.Fprintln(stderr, "error: use either --output or --replace, not both")
		return 2
	}

	pdf, inputPath, err := readInput(inputArg, stdin, stdinIsTTY)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	dest, useStdout, err := resolveOutput(inputPath, output, replace, stdoutIsTTY)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	client, err := sealclub.FromEnv()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	sealed, err := client.Seal(pdf)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if useStdout {
		if _, err := stdout.Write(sealed); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	}

	if err := os.WriteFile(dest, sealed, 0o644); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	fmt.Fprintln(stderr, dest)
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
	if ext == "" {
		return inputPath + ".sealed.pdf"
	}
	return inputPath + ".sealed.pdf"
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
