# Push GitHub Secrets Operator Helm Chart

This Helm chart installs the Push GitHub Secrets Operator in your Kubernetes cluster. This operator automatically syncs Kubernetes secrets and configmaps to GitHub repository secrets based on repository topics.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- GitHub API token with appropriate permissions

## Installing the Chart

Add the Helm repository:
```bash
helm repo add qalisa/push-github-secrets-operator https://github.io/Qalisa/push-github-secrets-operator
helm repo update
```

Install the chart:
```bash
helm install operator push-github-secrets-operator/operator \
  --set github.apiToken=your-github-token \
  --namespace push-github-secrets-operator \
  --create-namespace
```

## Upgrading

To upgrade the release:
```bash
helm upgrade operator qalisa/push-github-secrets-operator
```

## Uninstalling

To uninstall the release:
```bash
helm uninstall push-github-secrets-operator
```