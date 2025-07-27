# n8nctl

> ⚡ A lightweight CLI for managing [n8n](https://n8n.io) workflows declaratively with YAML.

`n8nctl` makes it easier to define, preview, and deploy n8n workflows via YAML files — helping you version control and automate your automation.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## ✨ Features

- Define workflows in clean, readable YAML
- Convert YAML to n8n-compatible JSON
- Deploy workflows to your self-hosted or cloud n8n instance
- Preview JSON output before deploying
- Scriptable and CI/CD-friendly

---

## 📦 Installation

### 🐧 Linux / 🍎 macOS

Download the latest binaries and checksums from the [Releases](https://github.com/brandon-kyle-bailey/n8nctl/releases) page:

```bash
# Linux amd64
curl -LO https://github.com/brandon-kyle-bailey/n8nctl/releases/download/v0.1.0/n8nctl-linux-amd64
curl -LO https://github.com/brandon-kyle-bailey/n8nctl/releases/download/v0.1.0/n8nctl-linux-amd64.sha256

# macOS amd64 (Intel)
curl -LO https://github.com/brandon-kyle-bailey/n8nctl/releases/download/v0.1.0/n8nctl-darwin-amd64
curl -LO https://github.com/brandon-kyle-bailey/n8nctl/releases/download/v0.1.0/n8nctl-darwin-amd64.sha256

# macOS arm64 (Apple Silicon)
curl -LO https://github.com/brandon-kyle-bailey/n8nctl/releases/download/v0.1.0/n8nctl-darwin-arm64
curl -LO https://github.com/brandon-kyle-bailey/n8nctl/releases/download/v0.1.0/n8nctl-darwin-arm64.sha256

# Verify checksums (optional but recommended)
sha256sum --check n8nctl-linux-amd64.sha256
sha256sum --check n8nctl-darwin-amd64.sha256
sha256sum --check n8nctl-darwin-arm64.sha256

# Make executable and install (example: Linux amd64)
chmod +x n8nctl-linux-amd64
sudo mv n8nctl-linux-amd64 /usr/local/bin/n8nctl
```
