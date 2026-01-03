# Scripts Directory

This directory contains utility scripts for development and deployment.

## Available Scripts

### `start.sh`
One-command script to start both backend and frontend services for local development.

**Usage:**
```bash
./scripts/start.sh
```

**What it does:**
- Checks for Go and Node.js installation
- Installs frontend dependencies if needed
- Starts the backend service (port 8080)
- Starts the frontend service (port 3000)

**Requirements:**
- Go 1.25.5+
- Node.js 20+
- Configuration file at `cmd/aiguide/aiguide.yaml`

## Adding New Scripts

When adding new scripts to this directory:
1. Make them executable: `chmod +x scripts/your-script.sh`
2. Add proper error handling and usage instructions
3. Document them in this README
4. Follow the existing naming conventions

## Script Categories

- **Development**: Scripts for local development and testing
- **Build**: Scripts for building and packaging
- **Deployment**: Scripts for deployment automation (if any)
- **Utilities**: Helper scripts for various tasks
