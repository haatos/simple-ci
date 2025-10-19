# SimpleCI

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

SimpleCI is a lightweight, self-hosted continuous integration (CI) application designed to replace complex tools like Jenkins with a streamlined, modern alternative. Built entirely in Go, it leverages [templ](https://github.com/a-h/templ) for server-side rendering and [HTMX](https://htmx.org/) for dynamic, interactive UI updates without heavy JavaScript frameworks. SimpleCI focuses on simplicity, security, and ease of use, allowing you to define pipelines that run on remote agents via SSH.

## Overview

SimpleCI consists of a central **controller** (the web app) that manages credentials, agents, and pipelines. Pipelines are defined in YAML and executed on **agents**; remote machines you connect to via SSH using stored **credentials** (private keys). The controller triggers builds on events like Git pushes (via webhooks) and provides a real-time dashboard for monitoring.

Key goals:

- Minimal dependencies and easy deployment as a single cross-compilable binary.
- Secure credential management with encrypted storage.
- Pipelines with simple syntax for stages containing multiple steps.

## Features

- **Credential Management**: Store SSH private keys securely for authenticating to agents.
- **Agent Orchestration**: Register remote machines as build agents; the controller uses SSH to connect to them to run pipelines.
- **Pipeline Definition**: YAML-based pipelines with stages, steps, and artifacts are read from your git repository.
- Scheduling: Schedule pipelines to run using _cron_ scheduling expressions for a specified branch.
- **Web Dashboard**: Intuitive UI powered by templ and HTMX for creating/editing resources and viewing build logs in real-time.
- **Webhook Integration**: Trigger builds from GitHub/GitLab on push or PR events.
- **Artifact Storage**: Download build artifacts directly from the UI.

## Technology Stack

- **Backend**: Go (100% Go codebase for the controller and agent runner).
- **Templating**: [github.com/a-h/templ](https://github.com/a-h/templ) for type-safe HTML components.
- **Frontend Interactivity**: [HTMX](https://HTMX.org/) for AJAX-driven updates without a build step.
- **Database**: SQLite.
- **Networking**: SSH for agent communication; HTTP/HTTPS for the web interface.

No Node.js, no Docker required. Just deploy as a single binary.

## Quick Start

### Prerequisites

- Go 1.21 or later.
- A remote machine (agent) with SSH access.

### Installation

1. Clone the repository:

   ```
   git clone https://github.com/haatos/simple-ci.git
   cd simple-ci
   ```

2. Build the binary:

   ```
   go build -o simpleci ./cmd/simpleci/main.go
   ```

3. Run the controller:
   ```
   ./simpleci
   ```
   The web UI will be available at `http://localhost:8080`.

### Configuration

Use environment variables or a .env file for setup:

- `SIMPLECI_HASH_KEY`: Encryption key. This is automatically generated, and saved into .env, if not found.
- `SIMPLECI_DB_PATH`: Path to SQLite DB (default: `./db.sqlite`).
- `SIMPLECI_PORT`: HTTP listen port (default: `8080`).
- `SIMPLECI_GITHUB_WEBHOOK_SECRET`: Optional secret for GitHub webhook validation.

For production, use a reverse proxy like Caddy or Nginx for HTTPS.

## Usage

### Creating Credentials

1. Navigate to _Credentials_ page from the side menu or home page.
2. Click 'Add credential'.
3. Fill in the form: username, description (optional) and SSH private key (PEM format).
4. Click 'Add'.
5. The key is encrypted at rest using your `SIMPLECI_SECRET_KEY`.

Credentials are referenced in agent configurations.

### Setting Up Agents

1. Navigate to _Agents_ page from the side menu or home page.
2. Click 'Add agent'
3. Fill in the form: select a credential, name, hostname (for SSH including port, defaulting to 22 if omitted), workspace and description (optional).
4. Click 'Add'
5. Click the 'Test connection' button of the _Agent_ to verify it is setup correctly.

Agents run pipelines in isolated workspaces and clean up after builds.

### Defining Pipelines

1. Navigate to _Pipelines_ page from the side menu or home page.
2. Click 'Add pipeline'.
3. Fill in the form: select an agent, enter a name and description (optional), enter a repository path, enter a script path (path to the pipeline script within the repository).
4. Click 'Add'.
5. Click 'Run pipeline' and fill in branch (git repository branch) to test. You will be navigated to the pipeline run page.

Pipelines should follow the following format:

```yaml
stages:
  - stage: Test
    steps:
      - step: Run tests
        script: go test ./...
  - stage: Build
    steps:
      - step: Run build
        script: go build -o bin/simpleci cmd/simpleci/main.go
    artifacts: bin
```

## Contributing

1. Fork the repo and create a feature branch (`git checkout -b feature/my-feature`).
2. Commit your changes (`git commit -m 'Add my feature'`).
3. Push to the branch (`git push origin feature/my-feature`).
4. Open a Pull Request.

We welcome contributions for new step types, UI improvements, or integrations. Run tests with `go test ./...`.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
