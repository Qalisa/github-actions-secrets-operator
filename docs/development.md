# Local Development Guide

This guide explains how to set up your local development environment and run tests for the GitHub Secrets Operator.

## Prerequisites

The following tools are required:
- Docker
- VSCode with Go extension
- Homebrew (for macOS)

All other tools will be installed automatically through VSCode tasks:
- Go 1.21 or later
- kubectl
- kind (Kubernetes in Docker)
- golangci-lint (for code linting)
- entr (for watch mode)
- Kubebuilder

To install all prerequisites:

1. First, ensure Docker is installed and running
2. Install VSCode Go extension
3. Run the setup task:
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

### Testing Tasks
- `Run Unit Tests`: Run all unit tests
- `Run Specific Test`: Run a specific test by name
- `Run E2E Tests`: Run end-to-end tests
- `Generate Test Coverage`: Generate and view test coverage report
- `Watch Tests`: Automatically run tests when files change
- `Run Linter`: Run code linting checks

## Debugging

VSCode launch configurations are provided for debugging:

1. Debug Operator: Launches the operator with debugger attached
2. Debug Unit Tests: Debug the current test file
3. Debug E2E Tests: Debug the E2E test suite

To use:
1. Set breakpoints in your code
2. Press F5 or select Run > Start Debugging
3. Choose the appropriate debug configuration

## Testing with Sample Resources

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

3. Run tests:
```bash
# Using VSCode Command Palette:
> Tasks: Run Task > Run Unit Tests
```

4. Test with sample resources
5. Debug if needed using the provided launch configurations

## Cleanup

To clean up your development environment:
```bash
# Using VSCode Command Palette:
> Tasks: Run Task > Clean Development Environment
```

## Continuous Testing

For continuous testing during development:
```bash
# Using VSCode Command Palette:
> Tasks: Run Task > Watch Tests
```

This will automatically run tests when files change.

## Common Issues

1. If the Kind cluster creation fails, ensure Docker is running and you have sufficient permissions.
2. If tests fail with KUBEBUILDER_ASSETS errors, ensure you've run `make test` at least once to download the required assets.
3. For debugging connection issues, check your KUBECONFIG is correctly pointing to the Kind cluster.
