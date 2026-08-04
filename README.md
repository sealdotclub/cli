# sealclub CLI

Command-line client for [seal.club](https://seal.club).

```bash
brew tap sealdotclub/tap
brew trust sealdotclub/tap   # Homebrew 6+
brew install sealclub
export SEAL_API_KEY=sk_test_...
```

```bash
# file → file (default: doc.pdf.sealed.pdf)
sealclub doc.pdf
sealclub doc.pdf --output sealed.pdf

# pipes
cat doc.pdf | sealclub > sealed.pdf
sealclub doc.pdf > sealed.pdf

# in-place
sealclub doc.pdf --replace

# suppress spinner / status
sealclub doc.pdf --quiet
```

Optional: `SEAL_API_BASE_URL` (default `https://api.seal.club`).


## Install from source

```bash
go install github.com/sealdotclub/cli/cmd/sealclub@latest
```

## License

Apache-2.0
