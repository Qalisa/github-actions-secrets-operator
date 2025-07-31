# GitHub Actions Secrets Operator

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

## GitHub App Setup

1. Create a new GitHub App:
   - Go to your organization's settings
   - Navigate to Developer Settings > GitHub Apps
   - Click "New GitHub App"

2. Configure the app:
   - Name: Choose an unique, descriptive name (e.g., "My K8s Secrets Syncer Operator")
   - Homepage URL: Your organization URL (only descriptive)
   - Webhook: Check to `Disable` (not needed)
   - Permissions:
     - Repository permissions:
       - `Actions secrets`: Read and write
       - `Actions variables`: Read and write

3. Generate and download a private key; we'll feed it to Helm. 

4. Get the `AppID` from the Settings page of your Github App, we'll feed it to Helm.

5. Install the app in your organization, then keep in mind the `InstallationID` what was generated for you by looking at this installation page URL; we'll feed it to Helm.

### Using Helm

1. Add the Helm repository:
```bash
helm repo add qalisa https://qalisa.github.io/charts
helm repo update
```

2. Install the operator:
```bash
helm install github-actions-secrets-operator qalisa/github-actions-secrets-operator \
  --set github.appId=<your-app-id> \
  --set github.installationId=<your-installation-id> \
  --set github.privateKey.explicit="$(cat path/to/private-key.pem)"
```

Or using an existing secret:
```bash
helm install github-actions-secrets-operator qalisa/github-actions-secrets-operator \
  --set github.appId=<your-app-id> \
  --set github.installationId=<your-installation-id> \
  --set github.privateKey.existingSecret=my-github-secret
```

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
    - secretRef: 
        name: db-credentials
        namespace: special
      key: DB_PASSWORD
      # githubSecretName defaults to key if not set
    - secretRef: 
        name: api-credentials
        namespace: special
      key: API_KEY
      githubSecretName: CUSTOM_API_KEY
  variables:
    - configMapRef: 
        name: env-config
        namespace: specific-app
      key: ENVIRONMENT
      # githubVariableName defaults to key if not set
    - configMapRef: 
        name: region-config
        namespace: specific-app
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
  repository: "MyOrganization/my-repository"
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

For detailed instructions on setting up your development environment and debugging, please see our [Development Guide](docs/development.md).

### Prerequisites

- Docker
- VSCode with Go extension
- Homebrew (for macOS)

All other dependencies (Go, kubectl, kind, etc.) will be installed automatically through VSCode tasks.

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/Qalisa/github-actions-secrets-operator.git
cd github-actions-secrets-operator
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

Apache License 2.0
