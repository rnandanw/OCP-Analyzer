# Quick Start Guide - OpenShift Analyzer

## 60-Second Setup

```bash
# 1. Install prerequisites
pip install o-must-gather
sudo dnf install jq   # or: sudo apt install jq

# 2. Build the tool
go build -o openshift-analyzer openshift-analyzer.go

# 3. Run analysis
./openshift-analyzer /path/to/must-gather
```

## Most Common Commands

```bash
# Quick health check
./openshift-analyzer -mode health /path/to/must-gather

# Find what is broken
./openshift-analyzer -mode issues /path/to/must-gather

# Full analysis with troubleshooting tips
./openshift-analyzer -verbose /path/to/must-gather

# Generate text report for support case
./openshift-analyzer -no-color /path/to/must-gather > report.txt
```

## Understanding Output Status

### Status Indicators

| Indicator | Meaning | Action Required |
|-----------|---------|-----------------|
| `[OK]` | Everything is healthy | No action needed |
| `[WARNING]` | Potential issues | Review recommendations, may need attention |
| `[ERROR]` | Critical problems | Immediate attention required |
| `[INFO]` | Informational | Context and explanations |

### Color Coding (when enabled)

- Green text: OK status
- Yellow text: WARNING status
- Red text: ERROR status
- Blue text: INFO status

Note: Colors can be disabled with `-no-color` flag for better accessibility or when redirecting to files.

## Key Sections to Review

1. **ETCD Endpoint Health** - Must all be healthy for cluster stability
2. **Degraded Operators** - Any non-True status needs investigation
3. **Degraded Nodes** - Nodes must be in Ready state
4. **Failing Pods** - Investigate CrashLoopBackOff, ImagePullBackOff
5. **MCP Status** - Should be Updated=True, Updating=False, Degraded=False

## Critical Issues (Act Immediately)

- **ETCD unhealthy** - Cluster at risk
- **Multiple operators degraded** - Core functionality impacted
- **Master nodes NotReady** - Control plane compromised
- **MCP degraded** - Cannot update cluster

## Pro Tips

### Before Opening Support Case

```bash
# Generate complete report with troubleshooting
./openshift-analyzer -verbose /path/to/must-gather > analysis.txt

# Attach analysis.txt to support case
```

### Regular Health Checks

```bash
# Collect must-gather
oc adm must-gather --dest-dir=/tmp/mg-$(date +%Y%m%d)

# Run health analysis
./openshift-analyzer -mode health /tmp/mg-*
```

### Finding Specific Issues

```bash
# Only degraded components
./openshift-analyzer -mode issues /path/to/must-gather

# With detailed troubleshooting
./openshift-analyzer -mode issues -verbose /path/to/must-gather
```

## Common Problems and Solutions

### Tool Will Not Run

```bash
# Check prerequisites
which omg jq column go

# If missing, install them (see README)
```

### Must-gather not found

```bash
# Verify path
ls -la /path/to/must-gather/

# Should see directories like:
# quay-io-openshift-release-dev-...
```

### No Output or Errors

```bash
# Manually set omg context
omg use /path/to/must-gather

# Verify
omg get nodes

# Then run analyzer again
```

## More Information

- Full documentation: `README-analyzer.md`
- OpenShift docs: https://docs.openshift.com
- Red Hat support: https://access.redhat.com
- IBM Design Language: https://www.ibm.com/design/language/

## Understanding Modes

| Mode | When to Use | Output |
|------|-------------|--------|
| `health` | Pre-upgrade checks, routine audits | Infrastructure, ETCD, configs, all statuses |
| `issues` | Incident response, finding problems | Only degraded/failed components |
| `full` | Complete analysis, support cases | Everything (health + issues) |

## Getting Help

1. **Tool Issues**: Check README troubleshooting section
2. **Cluster Issues**: Review tool recommendations
3. **Production Problems**: Open Red Hat support case with analysis report

---

**Remember**: Run with `-verbose` flag for detailed troubleshooting guidance

**Accessibility**: Use `-no-color` flag when redirecting output or for better screen reader compatibility
