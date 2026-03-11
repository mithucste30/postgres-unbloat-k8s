# Helm Repository Setup

This document explains how to set up and use the Helm chart repository for postgres-unbloat-k8s.

## Automated Publishing

The Helm chart is automatically published to GitHub Pages whenever a new version tag is pushed.

### What Happens on Release

When you push a tag (e.g., `v0.6.0`), the CI/CD pipeline:

1. **Builds Docker image** and pushes to GHCR
2. **Packages Helm chart** as a `.tgz` file
3. **Uploads to GitHub Releases**
4. **Generates Helm repository index** (`index.yaml`)
5. **Publishes to GitHub Pages** via `gh-pages` branch

### Prerequisites

To enable automatic Helm publishing, you need to:

#### 1. Enable GitHub Pages

1. Go to your repository on GitHub
2. Click **Settings** > **Pages**
3. Under **Source**, select:
   - **Source**: Deploy from a branch
   - **Branch**: `gh-pages`
   - **Directory**: `/ (root)`
4. Click **Save**

#### 2. Create Initial gh-pages Branch (Optional)

The first release will automatically create the `gh-pages` branch, but you can create it manually:

```bash
git checkout --orphan gh-pages
git rm -rf .
echo "# postgres-unbloat-k8s Helm Repository" > README.md
git add README.md
git commit -m "Initialize GitHub Pages"
git push origin gh-pages
```

## Using the Helm Repository

### Add the Repository

```bash
helm repo add mithucste30 https://mithucste30.github.io/postgres-unbloat-k8s
helm repo update
```

### Search Available Versions

```bash
helm search repo mithucste30/postgres-unbloat-k8s
```

### Install Specific Version

```bash
helm install postgres-unbloat mithucste30/postgres-unbloat-k8s \
  --version v0.5.0 \
  --namespace postgres-test \
  --create-namespace
```

### Install Latest Version

```bash
helm install postgres-unbloat mithucste30/postgres-unbloat-k8s \
  --namespace postgres-test \
  --create-namespace
```

### Install with Custom Values

```bash
helm install postgres-unbloat mithucste30/postgres-unbloat-k8s \
  --namespace postgres-test \
  --create-namespace \
  --set config.dryRun=false \
  --set monitoring.enabled=true \
  --set monitoring.namespace=monitoring
```

### Upgrade Existing Installation

```bash
helm upgrade postgres-unbloat mithucste30/postgres-unbloat-k8s \
  --namespace postgres-test
```

## Manual Installation (Alternative)

If GitHub Pages is not set up, you can install directly from the Git repository:

```bash
helm install postgres-unbloat deploy/helm \
  --namespace postgres-test \
  --create-namespace
```

Or from GitHub Releases:

```bash
# Download the chart
gh release download v0.5.0 --pattern "*.tgz"

# Install from downloaded archive
helm install postgres-unbloat ./postgres-unbloat-k8s-0.5.0.tgz \
  --namespace postgres-test \
  --create-namespace
```

## CI/CD Workflow

The release workflow `.github/workflows/release.yml` contains two Helm-related jobs:

### 1. Package and Publish Helm Chart

- Packages the Helm chart from `deploy/helm/`
- Creates versioned `.tgz` file
- Uploads to GitHub Releases
- Runs on every tag push (e.g., `v*.*.*`)

### 2. Publish Helm Repository Index

- Downloads the chart package from GitHub Releases
- Generates `index.yaml` using `helm repo index`
- Updates `gh-pages` branch
- Publishes to GitHub Pages
- Depends on successful chart packaging

## Troubleshooting

### Helm Repository Not Found

If you get a 404 error when adding the repository:

1. **Check GitHub Pages is enabled**
   ```bash
   # Check if gh-pages branch exists
   git ls-remote --heads origin gh-pages
   ```

2. **Check index.yaml exists**
   ```bash
   curl -I https://mithucste30.github.io/postgres-unbloat-k8s/index.yaml
   ```

3. **Wait for GitHub Pages to deploy**
   - First deployment can take 1-2 minutes
   - Check repository Settings > Pages for deployment status

### Chart Installation Fails

If installation fails:

1. **Verify chart version exists**
   ```bash
   helm search repo mithucste30/postgres-unbloat-k8s -l
   ```

2. **Update repository index**
   ```bash
   helm repo update mithucste30
   ```

3. **Check GitHub Release**
   - Visit: https://github.com/mithucste30/postgres-unbloat-k8s/releases
   - Verify the `.tgz` file is attached

### Index Not Updating

If the Helm index is not updating:

1. **Check Actions tab**
   - Go to repository **Actions** tab
   - Look for failed runs in the "Release" workflow
   - Check the "Publish Helm Repository Index" job logs

2. **Verify gh-pages branch**
   ```bash
   git fetch origin gh-pages
   git checkout gh-pages
   ls -la  # Should contain index.yaml and .tgz files
   ```

3. **Re-run the workflow**
   - Go to the failed workflow run
   - Click "Re-run all jobs"

## Repository URL

Once GitHub Pages is enabled, the Helm repository will be available at:

```
https://mithucste30.github.io/postgres-unbloat-k8s
```

This URL will serve the `index.yaml` file that Helm uses to discover available chart versions.

## Best Practices

### Version Management

- Use semantic versioning for tags (e.g., `v0.6.0`, `v0.7.0`)
- Update `Chart.yaml` appVersion when tagging
- Keep CHANGELOG updated with chart changes

### Testing Before Release

1. Test chart installation locally:
   ```bash
   helm install test-release deploy/helm --dry-run --debug
   ```

2. Test with specific values:
   ```bash
   helm template test deploy/helm --set monitoring.enabled=true
   ```

3. Lint the chart:
   ```bash
   helm lint deploy/helm
   ```

### Publishing Checklist

Before pushing a release tag:

- [ ] All tests pass locally
- [ ] CI/CD pipeline passes
- [ ] Chart version updated in `deploy/helm/Chart.yaml`
- [ ] Values documented in `deploy/helm/values.yaml`
- [ ] CHANGELOG updated
- [ ] Release notes prepared

## Related Documentation

- [Main README](../README.md)
- [Helm Chart Values](./helm/VALUES.md) (if available)
- [CI/CD Documentation](./CI.md) (if available)
