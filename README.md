# sealclub CLI

Command-line client for [seal.club](https://seal.club).

```bash
brew tap sealdotclub/tap
brew install sealclub
export SEAL_API_KEY=sk_test_...
```

```bash
# file → file (default: doc.pdf.sealed.pdf)
sealclub doc.pdf
sealclub doc.pdf -o sealed.pdf

# pipes
cat doc.pdf | sealclub > sealed.pdf
sealclub doc.pdf > sealed.pdf

# in-place
sealclub doc.pdf --replace
```

Optional: `SEAL_API_BASE_URL` (default `https://api.seal.club`).

## Install from source

```bash
go install github.com/sealdotclub/cli/cmd/sealclub@latest
```

## License

Apache-2.0
