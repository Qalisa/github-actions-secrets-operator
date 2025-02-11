# Push GitHub Secrets Operator

A Kubernetes operator to synchronize secrets and variables to GitHub repositories, bypassing GitHub Free Plan restrictions on organization-level secrets.

## Overview

This operator allows you to manage GitHub Actions secrets and variables at a repository level using Kubernetes resources. It's particularly useful for organizations using GitHub's Free Plan, which doesn't include organization-level secrets.

Key features:
- Sync Kubernetes Secrets to GitHub Actions secrets
- Sync ConfigMap values to GitHub Actions variables
- Cluster-scoped resources for organization-wide management
- Automatic synchronization on changes
- Rate limiting handling
- Status conditions for monitoring

## Installation

### Prerequisites

- Kubernetes cluster 1.19+
- Helm 3.0+
- GitHub App credentials (see setup below)

### Using Helm

1. Add the Helm repository:
```bash
helm repo add qalisa https://qalisa.github.io/charts
helm repo update
```

2. Install the operator:
```bash
helm install push-github-secrets-operator qalisa/push-github-secrets-operator \
  --set github.appId=<your-app-id> \
  --set github.installationId=<your-installation-id> \
  --set github.privateKey="$(cat path/to/private-key.pem)"
```

Or using an existing secret:
```bash
helm install push-github-secrets-operator qalisa/push-github-secrets-operator \
  --set github.appId=<your-app-id> \
  --set github.installationId=<your-installation-id> \
  --set github.existingSecret=my-github-secret
```

## GitHub App Setup

1. Create a new GitHub App:
   - Go to your organization's settings
   - Navigate to Developer Settings > GitHub Apps
   - Click "New GitHub App"

2. Configure the app:
   - Name: Choose a descriptive name (e.g., "K8s Secrets Sync")
   - Homepage URL: Your organization URL
   - Webhook: Disable (not needed)
   - Permissions:
     - Repository permissions:
       - Actions secrets and variables: Read and write

3. Generate and download the private key

4. Install the app in your organization

5. Note down:
   - App ID (from the app's settings page)
   - Installation ID (from the installation URL or API)
   - Private key (downloaded in step 3)

## Usage

### 1. Define Secret/Variable Groups

Create a `GithubActionSecretsSync` resource to define which secrets and variables should be synchronized:

```yaml
apiVersion: qalisa.github.io/v1alpha1
kind: GithubActionSecretsSync
metadata:
  name: prod-secrets
spec:
  secrets:
    - secretRef: db-credentials
      key: DB_PASSWORD
      # githubSecretName defaults to key if not set
    - secretRef: api-credentials
      key: API_KEY
      githubSecretName: CUSTOM_API_KEY
  variables:
    - configMapRef: env-config
      key: ENVIRONMENT
      # githubVariableName defaults to key if not set
    - configMapRef: region-config
      key: REGION
      githubVariableName: CUSTOM_REGION
```

### 2. Bind Repositories

Create a `GithubSyncRepo` resource to specify which repositories should receive which secrets/variables:

```yaml
apiVersion: qalisa.github.io/v1alpha1
kind: GithubSyncRepo
metadata:
  name: my-repo-sync
spec:
  repository: "Qalisa/my-repository"
  secretsSyncRefs:
    - prod-secrets
    - staging-secrets
```

### 3. Monitor Status

Check the status of your resources:

```bash
kubectl get githubactionsecretssyncs
kubectl get githubsyncrepoes
```

## Development

For detailed instructions on setting up your development environment, running tests, and debugging, please see our [Development Guide](docs/development.md).

### Prerequisites

- Docker
- VSCode with Go extension
- Homebrew (for macOS)

All other dependencies (Go, kubectl, kind, etc.) will be installed automatically through VSCode tasks.

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/Qalisa/push-github-secrets-operator.git
cd push-github-secrets-operator
```

2. Set up development environment:
   ```bash
   # Using VSCode Command Palette (Cmd/Ctrl + Shift + P):
   > Tasks: Run Task > Setup Development Environment
   ```

3. Start local development:
   ```bash
   # Using VSCode Command Palette:
   > Tasks: Run Task > Start Development Environment
   ```

This will create a local Kind cluster and deploy the operator for testing.

### Available Tasks

- Run tests: `Tasks: Run Task > Run Unit Tests`
- Run E2E tests: `Tasks: Run Task > Run E2E Tests`
- Run linter: `Tasks: Run Task > Run Linter`
- Generate test coverage: `Tasks: Run Task > Generate Test Coverage`

See the [Development Guide](docs/development.md) for complete documentation.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

Apache License 2.0
