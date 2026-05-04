# OpenShift Must-Gather Comprehensive Analyzer

A powerful, all-in-one Go tool that combines cluster health analysis and issue identification for OpenShift must-gather bundles.

## 🌟 Features

### Dual Analysis Modes

1. **Health Analysis Mode** - Comprehensive cluster health assessment
   - Infrastructure and platform details
   - ETCD cluster health and status
   - Cluster version and upgrade history
   - Configuration review (install-config, proxy)
   - Operational status (operators, nodes, MCPs, machines)
   - Pod health and restart analysis
   - Control plane component logs

2. **Issue Identification Mode** - Focused problem detection
   - Degraded cluster operators with detailed analysis
   - Degraded machine config pools
   - Machines not in Running state
   - Nodes not in Ready state
   - Failing pods analysis
   - Machine-config-daemon logs for degraded nodes

3. **Full Analysis Mode** - Combined health + issues (default)

### Enhanced Features

✅ **Rich Troubleshooting Guidance** - Context-aware recommendations for every issue  
✅ **Color-Coded Output** - Visual indicators for status (✓ ✗ ⚠ ℹ)  
✅ **Verbose Mode** - Detailed explanations and troubleshooting steps  
✅ **Smart Detection** - Automatic identification of common problems  
✅ **Comprehensive Checks** - Configuration drift, resource issues, component health  
✅ **Actionable Recommendations** - Specific commands and next steps  

## 📋 Prerequisites

### Required Tools

1. **omg** (o-must-gather) - OpenShift must-gather analyzer
   ```bash
   pip install o-must-gather
   # Verify: omg --version
   ```

2. **jq** - JSON processor
   ```bash
   # Red Hat/Fedora
   sudo dnf install jq
   
   # Debian/Ubuntu
   sudo apt install jq
   
   # MacOS
   brew install jq
   ```

3. **column** - Text formatting utility (usually pre-installed)
   ```bash
   # Red Hat/Fedora
   sudo dnf install util-linux
   
   # Debian/Ubuntu
   sudo apt install bsdmainutils
   ```

4. **Go** 1.16+ (to compile)
   ```bash
   # Download from https://go.dev/dl/
   ```

## 🚀 Installation

### Option 1: Build from Source

```bash
# Clone or download the source
cd /path/to/source

# Build the binary
go build -o openshift-analyzer openshift-analyzer.go

# Make executable
chmod +x openshift-analyzer

# Optional: Install to system path
sudo mv openshift-analyzer /usr/local/bin/
```

### Option 2: Quick Run

```bash
# Run directly without building
go run openshift-analyzer.go [OPTIONS] <must-gather-directory>
```

## 📖 Usage

### Basic Usage

```bash
# Full analysis (default)
./openshift-analyzer /path/to/must-gather

# Health analysis only
./openshift-analyzer -mode health /path/to/must-gather

# Issue identification only
./openshift-analyzer -mode issues /path/to/must-gather

# Verbose output with troubleshooting
./openshift-analyzer -verbose /path/to/must-gather

# Disable colors (for piping to file)
./openshift-analyzer -no-color /path/to/must-gather > report.txt
```

### Command-Line Options

| Option | Values | Description |
|--------|--------|-------------|
| `-mode` | `health`, `issues`, `full` | Analysis mode (default: `full`) |
| `-verbose` | flag | Enable detailed troubleshooting guidance |
| `-no-color` | flag | Disable colored output |

### Usage Examples

```bash
# Quick health check
./openshift-analyzer -mode health /data/must-gather-2026-05-04

# Deep-dive issue analysis with troubleshooting
./openshift-analyzer -mode issues -verbose /data/must-gather-2026-05-04

# Complete analysis with all details
./openshift-analyzer -mode full -verbose /data/must-gather-2026-05-04

# Generate text report
./openshift-analyzer -no-color /data/must-gather-2026-05-04 > cluster-report.txt
```

## 📊 Analysis Sections

### Health Analysis Includes

1. **Cluster Infrastructure** - Platform type, topology, API endpoints
2. **ETCD Health** - Endpoint health, status, member list, quorum
3. **Cluster Version** - Current version, conditions, upgrade history
4. **Configuration** - Install config, proxy settings
5. **Operators** - Status of all cluster operators
6. **Nodes** - Node status, machine configuration drift
7. **Machine Config Pools** - MCP status and updates
8. **Machines/MachineSets** - Machine API resources
9. **Pods** - Failing pods and high restart counts
10. **Control Plane Logs** - kube-apiserver, ETCD, kube-controller-manager

### Issue Analysis Includes

1. **Degraded Operators** - Identification and detailed YAML analysis
2. **Degraded MCPs** - Machine config pool issues
3. **Machine Problems** - Machines not in Running state
4. **Node Issues** - Nodes not Ready, SchedulingDisabled
5. **Pod Failures** - Comprehensive pod failure analysis
6. **MCD Logs** - Machine-config-daemon logs for degraded nodes

## 🎯 Troubleshooting Features

### Automatic Detection

- ETCD health issues with quorum analysis
- Configuration drift (current vs desired configs)
- Resource exhaustion indicators
- High restart counts (>10 restarts)
- Large ETCD database warnings (>8GB)

### Context-Aware Recommendations

Each issue includes:
- **Root cause indicators** - What to look for
- **Diagnostic commands** - Specific oc/kubectl commands
- **Common solutions** - Known fixes and workarounds
- **Documentation links** - Relevant OpenShift docs
- **Best practices** - Prevention strategies

### Example Troubleshooting Output

```
⚠ Some ETCD endpoints are unhealthy!

ℹ Troubleshooting Steps:
  Check ETCD pod logs for errors
  Verify network connectivity between ETCD members
  Check master node resources (CPU, memory, disk)
  Review ETCD certificates and authentication
  Consult: https://docs.openshift.com/container-platform/latest/backup_and_restore/...
```

## 🔍 Common Use Cases

### 1. Pre-Upgrade Health Check

```bash
# Before upgrading, verify cluster health
./openshift-analyzer -mode health -verbose /path/to/must-gather
```

### 2. Incident Response

```bash
# Quickly identify what's broken
./openshift-analyzer -mode issues /path/to/must-gather
```

### 3. Support Case Preparation

```bash
# Generate comprehensive report for Red Hat support
./openshift-analyzer -verbose /path/to/must-gather > support-report.txt
```

### 4. Routine Cluster Audit

```bash
# Regular cluster health assessment
./openshift-analyzer -mode health /path/to/must-gather
```

## 📝 Output Examples

### Status Indicators

- ✓ (Green) - Healthy/Success
- ✗ (Red) - Failed/Error
- ⚠ (Yellow) - Warning/Degraded
- ℹ (Cyan) - Information

### Sample Output

```
╔══════════════════════════════════════════════════════════════╗
║   OpenShift Must-Gather Comprehensive Analyzer v2.0         ║
║   Cluster Health & Issue Detection Tool                     ║
╚══════════════════════════════════════════════════════════════╝

✓ omg command found
✓ jq command found
✓ column command found
✓ must-gather directory found: /data/must-gather-2026-05-04
✓ omg context initialized

▶ Cluster Infrastructure Details
────────────────────────────────────────────────────────────────
  apiServer: https://api.cluster.example.com:6443
  platform: AWS
  platformStatus:
    type: AWS
...
```

## 🐛 Troubleshooting the Tool

### Prerequisites Not Found

If you get errors about missing commands:

```bash
# Check what's installed
which omg jq column

# Install missing tools (see Prerequisites section)
```

### Must-Gather Path Issues

```bash
# Verify must-gather structure
ls -la /path/to/must-gather/

# Should contain timestamped directories like:
# quay-io-openshift-release-dev-ocp-v4-0-art-dev-sha256-...
```

### OMG Context Errors

```bash
# Manually set omg context
omg use /path/to/must-gather

# Verify
omg get nodes
```

## 📄 License

Open source - feel free to modify and distribute

## 🔗 Useful Links

- [OpenShift Must-Gather](https://docs.openshift.com/container-platform/latest/support/gathering-cluster-data.html)
- [o-must-gather Tool](https://pypi.org/project/o-must-gather/)
- [OpenShift Troubleshooting](https://docs.openshift.com/container-platform/latest/support/troubleshooting/index.html)
- [ETCD Operations](https://docs.openshift.com/container-platform/latest/backup_and_restore/control_plane_backup_and_restore/disaster_recovery/about-disaster-recovery.html)

## 📞 Support

For issues with this tool:
- Check the troubleshooting section above
- Verify all prerequisites are installed
- Ensure must-gather is complete and valid

For OpenShift cluster issues:
- Review tool output and recommendations
- Consult OpenShift documentation
- Open Red Hat support case for production issues

---

**Version:** 2.0  
**Last Updated:** 2026-05-04  
**Compatibility:** OpenShift 4.x must-gather bundles
