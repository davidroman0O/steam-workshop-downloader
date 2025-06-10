# Workshop - Steam Workshop Downloader

A simple CLI tool to download Steam Workshop items using SteamCMD.

## Quick Start

### 1. Install SteamCMD
```bash
workshop install
```

### 2. Download Workshop Items

Just paste the workshop URL:
```bash
workshop download https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437
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
workshop download https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437
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
workshop download https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437 --output ./my-mods
```

### Download private items (requires Steam credentials)
```bash
workshop download https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437 --username myuser --password mypass
```

## Configuration

The tool stores configuration in `~/.workshop.yaml`. You can set default directories:

```yaml
download_dir: /path/to/your/downloads
steamcmd_dir: /path/to/steamcmd
```

## Examples

**Project Zomboid mod:**
```bash
workshop download https://steamcommunity.com/sharedfiles/filedetails/?id=2503622437
```

**Cities: Skylines asset:**
```bash
workshop download https://steamcommunity.com/sharedfiles/filedetails/?id=12345678
```

**Garry's Mod addon:**
```bash
workshop download https://steamcommunity.com/sharedfiles/filedetails/?id=87654321
```

## Directories

- **Configuration:** `~/.workshop/`
- **SteamCMD:** `~/.workshop/steamcmd/`
- **Downloads:** `~/Downloads/Steam-Workshop/`
- **Workshop content:** `~/.workshop/steamcmd/steamapps/workshop/content/`

## Commands

- `workshop install` - Install SteamCMD
- `workshop download <url|id>` - Download workshop item
- `workshop --help` - Show help
- `workshop --version` - Show version info

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

### Releasing
The project uses GitHub Actions for automated builds and releases:

1. **Create a tag**: `git tag v1.0.0`
2. **Push the tag**: `git push origin v1.0.0`
3. **GitHub Actions will**:
   - Build binaries for all platforms
   - Create a GitHub release
   - Attach all binaries to the release

## Requirements

- Go 1.23+ (for building)
- Internet connection
- ~500MB disk space for SteamCMD 