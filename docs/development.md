# Local Development Guide

This guide explains how to set up your local development environment for the GitHub Secrets Operator.

## Prerequisites

The following tools are required:
- Docker
- VSCode with Go extension
- Homebrew (for macOS)
- Helm (for deployment)

All other tools will be installed automatically through VSCode tasks:
- Go 1.21 or later
- kubectl
- kind (Kubernetes in Docker)
- golangci-lint (for code linting)

To install all prerequisites:

1. First, ensure Docker is installed and running
2. Install VSCode Go extension
3. Install Helm:
```bash
# macOS
brew install helm

# Linux
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

4. Run the setup task:
```bash
# Using VSCode Command Palette (Cmd/Ctrl + Shift + P):
> Tasks: Run Task > Setup Development Environment
```

This will install all required tools and set up your development environment.

## Development Environment Setup

The easiest way to get started is to use the combined task:
```bash
# Using VSCode Command Palette (Cmd/Ctrl + Shift + P):
> Tasks: Run Task > Start Development Environment
```

This will:
1. Install all required tools
2. Create a local Kubernetes cluster
3. Deploy the operator for testing

## Available Tasks

The following tasks are available in VSCode:

### Main Tasks
- `Setup Development Environment`: Install all required tools
- `Start Development Environment`: Create cluster and deploy operator
- `Clean Development Environment`: Clean up all resources
- `Run Linter`: Run code linting checks

## Local Deployment

To deploy the operator locally:

1. Install CRDs:
```bash
make install
```

2. Deploy the operator:
```bash
make deploy
```

Note: If you make changes to the API (in api/v1alpha1/), you'll need to regenerate the CRDs:
```bash
make generate-crds
git add helm/push-github-secrets-operator/crds/
git commit -m "Update CRDs"
```

## Sample Resources

Sample resources are provided in `config/samples/`:

1. Create a test secret:
```bash
kubectl apply -f config/samples/test-secret.yaml
```

2. Create a GithubActionSecretsSync resource:
```bash
kubectl apply -f config/samples/test-secretsync.yaml
```

## Development Workflow

1. Make code changes
2. Run linter:
```bash
# Using VSCode Command Palette:
> Tasks: Run Task > Run Linter
```

3. Build and deploy:
```bash
make docker-build
make deploy
```

4. Test with sample resources

## Cleanup

To clean up your development environment:
```bash
# Using VSCode Command Palette:
> Tasks: Run Task > Clean Development Environment
```

## Common Issues

1. If the Kind cluster creation fails, ensure Docker is running and you have sufficient permissions.
2. For debugging connection issues, check your KUBECONFIG is correctly pointing to the Kind cluster.
