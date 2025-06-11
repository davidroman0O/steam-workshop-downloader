# Workshop - Steam Workshop Downloader

A simple CLI tool to download Steam Workshop items using SteamCMD.

## Quick Start

### 1. Install SteamCMD
```bash
workshop install
```

### 2. Login to Steam (for private/restricted items)
```bash
workshop login
```

### 3. Download Workshop Items

Just paste the workshop URL:
```bash
workshop download 'https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437'
```

Or use your Steam credentials for private items:
```bash
workshop download 'https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437' --username yourusername
```

That's it! The tool will automatically:
- Extract the App ID from the workshop page
- Download the item using SteamCMD
- Show you where it's stored

## Installation

### Download Pre-built Binaries
Go to [Releases](https://github.com/davidroman0O/steam-workshop-downloader/releases) and download the binary for your platform:

- **Linux**: `workshop-linux-amd64`, `workshop-linux-arm64`, `workshop-linux-386`, `workshop-linux-armv7`
- **macOS**: `workshop-darwin-amd64` (Intel), `workshop-darwin-arm64` (Apple Silicon)  
- **Windows**: `workshop-windows-amd64.exe`, `workshop-windows-386.exe`, `workshop-windows-arm64.exe`

### Build from source
```bash
git clone https://github.com/davidroman0O/steam-workshop-downloader
cd steam-workshop-downloader
make build
```

Or with Go directly:
```bash
go build -o workshop .
```

## Usage

### Install SteamCMD
```bash
workshop install
```

### Download Workshop Items

**From URL (easiest):**
```bash
workshop download 'https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437'
```

**From Workshop ID (requires App ID):**
```bash
workshop download 2503622437 --app-id 108600
```

**From App ID + Workshop ID:**
```bash
workshop download 108600 2503622437
```

### Extract to custom directory
```bash
workshop download 'https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437' --output ./my-mods
```

### Download private/restricted items

First, log into Steam interactively (handles Steam Guard codes):
```bash
workshop login
```

Then download using your cached credentials:
```bash
workshop download 'https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437' --username yourusername
```

The login command will:
- Prompt for your Steam username and password
- Handle Steam Guard 2FA codes automatically
- Cache your credentials for future downloads

## Configuration

The tool stores configuration in `~/.workshop.yaml`. You can set default directories:

```yaml
download_dir: /path/to/your/downloads
steamcmd_dir: /path/to/steamcmd
```

## Examples

**Project Zomboid mod:**
```bash
workshop download 'https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437'
```

**Cities: Skylines asset:**
```bash
workshop download 'https://steamcommunity.com/sharedfiles/filedetails/?id=12345678'
```

**Garry's Mod addon:**
```bash
workshop download 'https://steamcommunity.com/sharedfiles/filedetails/?id=87654321'
```

## Directories

- **Configuration:** `~/.workshop/`
- **SteamCMD:** `~/.workshop/steamcmd/`
- **Downloads:** `~/Downloads/Steam-Workshop/`
- **Workshop content:** `~/.workshop/steamcmd/steamapps/workshop/content/`

## Commands

- `workshop install` - Install SteamCMD
- `workshop login` - Log into Steam (interactive, handles Steam Guard)
- `workshop download <url|id>` - Download workshop item
- `workshop clean` - Clean workshop cache (fixes SteamCMD errors)
- `workshop --help` - Show help
- `workshop --version` - Show version info

## Troubleshooting

### CWorkThreadPool Errors

Sometimes SteamCMD may fail with errors like:
```
CWorkThreadPool::~CWorkThreadPool: work complete queue not empty, 1 items discarded.
CWorkThreadPool::~CWorkThreadPool: work processing queue not empty: 1 items discarded.
```

**Quick Fix:**
```bash
workshop clean
```

This command removes SteamCMD's workshop cache directories that can become corrupted and cause these errors. The clean command will:

- Remove workshop downloads cache
- Remove workshop temp files  
- Preserve your downloaded content (unless you use `--all`)

**Options:**
```bash
workshop clean           # Clean cache only (recommended)
workshop clean --all     # Also remove downloaded workshop content
workshop clean --force   # Skip confirmation prompt
```

After cleaning, try your download again. This fixes most SteamCMD hanging/error issues.

### Intermittent Download Failures

Sometimes downloads may fail with various errors like "Failure", "No subscription", or network timeouts, even when:
- Your authentication is working correctly
- The workshop item exists and is accessible
- Your internet connection is stable

**This is normal behavior** - Steam servers can be temperamental. The failure might resolve itself anywhere from a few minutes to a few hours later.

**What to try:**
1. **Wait and retry** - Often the same command works perfectly 10-30 minutes later
2. **Try different times** - Steam servers have varying load throughout the day
3. **Check if it's a specific item** - Try downloading a different workshop item to see if it's server-wide

**Example of typical behavior:**
```bash
# First attempt - fails
$ workshop download 'https://steamcommunity.com/sharedfiles/filedetails/?id=123456' --username myuser
ERROR! Download item 123456 failed (Failure).

# Same command 20 minutes later - works perfectly  
$ workshop download 'https://steamcommunity.com/sharedfiles/filedetails/?id=123456' --username myuser
Successfully downloaded to: [...]/123456
```

The retry system with Fibonacci backoff will automatically retry failed downloads, but for persistent issues, manual retries after waiting often succeed.

## Development

### Building locally
```bash
# Simple build
make build

# Build for all platforms
make build-all

# Development build (faster)
make dev

# Clean build artifacts
make clean
```

## Requirements

- Go 1.23+ (for building)
- Internet connection
- ~500MB disk space for SteamCMD 