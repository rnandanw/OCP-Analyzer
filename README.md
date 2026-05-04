# OpenShift Must-Gather Comprehensive Analyzer

A professional, accessible Go tool that combines cluster health analysis and issue identification for OpenShift must-gather bundles.

## Features

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

- **Professional Output** - Clean, text-based status indicators [OK], [ERROR], [WARNING], [INFO]
- **Accessible Design** - No symbol dependencies, screen-reader friendly
- **Rich Troubleshooting** - Context-aware recommendations for every issue
- **Color-Coded Display** - Optional colored output (can be disabled with -no-color)
- **Verbose Mode** - Detailed explanations and troubleshooting steps
- **Smart Detection** - Automatic identification of common problems
- **Actionable Recommendations** - Specific commands and next steps

## Prerequisites

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

## Installation

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

## Usage

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

## Analysis Sections

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

## Output Format

### Status Indicators

The tool uses clear text-based status indicators:

- `[OK]` - Healthy/Success (displayed in green when color is enabled)
- `[ERROR]` - Failed/Critical issue (displayed in red when color is enabled)
- `[WARNING]` - Warning/Degraded state (displayed in yellow when color is enabled)
- `[INFO]` - Informational message (displayed in blue when color is enabled)

### Sample Output

```
================================================================
OpenShift Must-Gather Comprehensive Analyzer v2.0
Cluster Health & Issue Detection Tool
================================================================

Analysis Mode: full
Verbose Output: true
Must-Gather Path: /data/must-gather-2026-05-04

[INFO] Validating prerequisites...
[OK] omg command found
[OK] jq command found
[OK] column command found
[OK] must-gather directory found: /data/must-gather-2026-05-04
[OK] omg context initialized

================================================================
CLUSTER HEALTH ANALYSIS
================================================================

Cluster Infrastructure Details
--------------------------------------------------------------------------------
  apiServer: https://api.cluster.example.com:6443
  platform: AWS
  platformStatus:
    type: AWS
...
```

## Troubleshooting Features

### Automatic Detection

- ETCD health issues with quorum analysis
- Configuration drift (current vs desired configs)
- Resource exhaustion indicators
- High restart counts (more than 10 restarts)
- Large ETCD database warnings (greater than 8GB)

### Context-Aware Recommendations

Each issue includes:
- **Root cause indicators** - What to look for
- **Diagnostic commands** - Specific oc/kubectl commands
- **Common solutions** - Known fixes and workarounds
- **Documentation links** - Relevant OpenShift docs
- **Best practices** - Prevention strategies

### Example Troubleshooting Output

```
[WARNING] Some ETCD endpoints are unhealthy

[INFO] Troubleshooting Steps:
  1. Check ETCD pod logs for errors
  2. Verify network connectivity between ETCD members
  3. Check master node resources (CPU, memory, disk)
  4. Review ETCD certificates and authentication
  5. Consult: https://docs.openshift.com/container-platform/latest/backup_and_restore/...
```

## Common Use Cases

### 1. Pre-Upgrade Health Check

```bash
# Before upgrading, verify cluster health
./openshift-analyzer -mode health -verbose /path/to/must-gather
```

### 2. Incident Response

```bash
# Quickly identify what is broken
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

## Troubleshooting the Tool

### Prerequisites Not Found

If you get errors about missing commands:

```bash
# Check what is installed
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

## Design Principles

- **Clarity** - Clear, unambiguous text-based status indicators
- **Accessibility** - No symbol dependencies, works with screen readers
- **Professionalism** - Clean, structured output format
- **Consistency** - Uniform formatting and messaging throughout
- **Usability** - Intuitive command-line interface

## License

Open source - feel free to modify and distribute

## Useful Links

- [OpenShift Must-Gather](https://docs.openshift.com/container-platform/latest/support/gathering-cluster-data.html)
- [o-must-gather Tool](https://pypi.org/project/o-must-gather/)
- [OpenShift Troubleshooting](https://docs.openshift.com/container-platform/latest/support/troubleshooting/index.html)
- [ETCD Operations](https://docs.openshift.com/container-platform/latest/backup_and_restore/control_plane_backup_and_restore/disaster_recovery/about-disaster-recovery.html)

## Support

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
**Design Standard:** IBM Design Language
