package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
)

const (
	colorRed     = "\033[0;31m"
	colorGreen   = "\033[0;32m"
	colorYellow  = "\033[0;33m"
	colorPurple  = "\033[0;35m"
	colorCyan    = "\033[0;36m"
	colorReset   = "\033[0m"
	bold         = "\033[1m"
	regular      = "\033[0m"
	checkMark    = "✓"
	crossMark    = "✗"
	warningMark  = "⚠"
	infoMark     = "ℹ"
)

type Config struct {
	mustGatherPath string
	mode           string
	verbose        bool
	noColor        bool
}

type ETCDEndpointHealth struct {
	Endpoint string `json:"endpoint"`
	Health   bool   `json:"health"`
	Took     string `json:"took"`
}

type ETCDEndpointStatus struct {
	Endpoint string `json:"Endpoint"`
	Status   struct {
		Header struct {
			MemberID int64 `json:"member_id"`
			RaftTerm int64 `json:"raft_term"`
		} `json:"header"`
		Leader           int64  `json:"leader"`
		Version          string `json:"version"`
		DBSize           int64  `json:"dbSize"`
		RaftIndex        int64  `json:"raftIndex"`
		RaftAppliedIndex int64  `json:"raftAppliedIndex"`
	} `json:"Status"`
}

type ETCDMember struct {
	Name       string   `json:"name"`
	PeerURLs   []string `json:"peerURLs"`
	ClientURLs []string `json:"clientURLs"`
}

type ETCDMemberList struct {
	Members []ETCDMember `json:"members"`
}

type AnalysisResult struct {
	Section  string
	Status   string
	Message  string
	Issues   []string
	Warnings []string
}

var (
	cfg     Config
	results []AnalysisResult
)

func main() {
	if err := run(); err != nil {
		printError("Fatal error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	parseFlags()

	if err := validate(); err != nil {
		return err
	}

	printBanner()

	switch cfg.mode {
	case "health":
		runHealthAnalysis()
	case "issues":
		runIssueAnalysis()
	case "full":
		runFullAnalysis()
	default:
		runFullAnalysis()
	}

	printSummary()
	return nil
}

func parseFlags() {
	flag.StringVar(&cfg.mode, "mode", "full", "Analysis mode: health, issues, or full")
	flag.BoolVar(&cfg.verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&cfg.noColor, "no-color", false, "Disable colored output")
	flag.Parse()

	if flag.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	cfg.mustGatherPath = flag.Arg(0)
}

func printUsage() {
	fmt.Printf("%s%sOpenShift Must-Gather Analyzer%s\n\n", bold, colorCyan, regular)
	fmt.Println("USAGE:")
	fmt.Printf("  %s [OPTIONS] <must-gather-directory>\n\n", os.Args[0])
	fmt.Println("OPTIONS:")
	fmt.Println("  -mode string")
	fmt.Println("        Analysis mode: health, issues, or full (default: full)")
	fmt.Println("        - health: General cluster health and configuration")
	fmt.Println("        - issues: Focus on identifying problems and degraded components")
	fmt.Println("        - full:   Complete analysis (both health + issues)")
	fmt.Println("  -verbose")
	fmt.Println("        Enable verbose output with detailed troubleshooting")
	fmt.Println("  -no-color")
	fmt.Println("        Disable colored output")
	fmt.Println("\nEXAMPLES:")
	fmt.Printf("  %s /path/to/must-gather\n", os.Args[0])
	fmt.Printf("  %s -mode issues /path/to/must-gather\n", os.Args[0])
	fmt.Printf("  %s -mode health -verbose /path/to/must-gather\n", os.Args[0])
}

func validate() error {
	printInfo("Validating prerequisites...")

	if flag.NArg() == 0 {
		return fmt.Errorf("no must-gather directory supplied\nUSAGE: %s [OPTIONS] <must-gather-directory>", os.Args[0])
	}

	if flag.NArg() > 1 {
		return fmt.Errorf("only one must-gather directory should be provided\nUSAGE: %s [OPTIONS] <must-gather-directory>", os.Args[0])
	}

	if !commandExists("omg") {
		return fmt.Errorf("%s omg command not found!\n\n%sTroubleshooting:%s\n"+
			"  1. Install o-must-gather: pip install o-must-gather\n"+
			"  2. Verify installation: omg --version\n"+
			"  3. Visit: https://pypi.org/project/o-must-gather\n"+
			"  4. For Python issues: ensure pip is installed (python3 -m pip --version)",
			crossMark, colorYellow, colorReset)
	}
	printSuccess("omg command found")

	if !commandExists("jq") {
		return fmt.Errorf("%s jq command not found!\n\n%sTroubleshooting:%s\n"+
			"  Red Hat/Fedora:  sudo dnf install jq\n"+
			"  Debian/Ubuntu:   sudo apt install jq\n"+
			"  MacOS:           brew install jq\n"+
			"  Manual install:  https://stedolan.github.io/jq/download",
			crossMark, colorYellow, colorReset)
	}
	printSuccess("jq command found")

	if !commandExists("column") {
		return fmt.Errorf("%s column command not found!\n\n%sTroubleshooting:%s\n"+
			"  Red Hat/Fedora:  sudo dnf install util-linux\n"+
			"  Debian/Ubuntu:   sudo apt install bsdmainutils\n"+
			"  Note: Usually pre-installed on most Linux distributions",
			crossMark, colorYellow, colorReset)
	}
	printSuccess("column command found")

	if _, err := os.Stat(cfg.mustGatherPath); os.IsNotExist(err) {
		return fmt.Errorf("%s must-gather directory does not exist: %s\n\n%sTroubleshooting:%s\n"+
			"  1. Verify the path is correct\n"+
			"  2. Ensure you have read permissions\n"+
			"  3. Check if the must-gather was extracted properly",
			crossMark, cfg.mustGatherPath, colorYellow, colorReset)
	}
	printSuccess(fmt.Sprintf("must-gather directory found: %s", cfg.mustGatherPath))

	configFile := filepath.Join(os.Getenv("HOME"), ".omgconfig")
	os.Remove(configFile)

	cmd := exec.Command("omg", "use", cfg.mustGatherPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s failed to set omg context: %v\n\nOutput: %s\n\n%sTroubleshooting:%s\n"+
			"  1. Verify must-gather structure is intact\n"+
			"  2. Check if must-gather was collected properly\n"+
			"  3. Try: omg use %s manually\n"+
			"  4. Ensure must-gather is uncompressed",
			crossMark, err, string(output), colorYellow, colorReset, cfg.mustGatherPath)
	}
	printSuccess("omg context initialized")

	fmt.Println()
	return nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func printBanner() {
	banner := `
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║   OpenShift Must-Gather Comprehensive Analyzer v2.0         ║
║   Cluster Health & Issue Detection Tool                     ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
`
	printColored(colorCyan, banner)
	fmt.Printf("Mode: %s%s%s | Verbose: %v | Must-Gather: %s\n\n",
		bold, cfg.mode, regular, cfg.verbose, filepath.Base(cfg.mustGatherPath))
}

func runFullAnalysis() {
	printHeader("COMPREHENSIVE ANALYSIS: HEALTH + ISSUES")
	runHealthAnalysis()
	fmt.Println("\n" + strings.Repeat("=", 80) + "\n")
	runIssueAnalysis()
}

func runHealthAnalysis() {
	printHeader("CLUSTER HEALTH ANALYSIS")

	sections := []struct {
		title string
		fn    func()
	}{
		{"Cluster Infrastructure Details", func() { infrastructure() }},
		{"ETCD Endpoint Health", func() { etcdEndpointHealth() }},
		{"ETCD Endpoint Status", func() { etcdEndpointStatus() }},
		{"ETCD Member List", func() { etcdMemberList() }},
		{"ClusterVersion Details", func() { clusterversion() }},
		{"Install-Config Configuration", func() { installConfigYAML() }},
		{"Cluster-Wide Proxy Configuration", func() { clusterWideProxy() }},
		{"Cluster Operators Status", func() { clusterOperator() }},
		{"Nodes Status", func() { nodes() }},
		{"Node Machine Configuration", func() { machineconfiguration() }},
		{"Machine Config Pool Status", func() { mcp() }},
		{"Machines Status", func() { machine() }},
		{"MachineSets Status", func() { machineset() }},
		{"Failing Pods", func() { pods() }},
		{"Pods with High Restart Count (>10)", func() { podRestart() }},
		{"Kube-APIServer Logs", func() { kubeApiserver() }},
		{"ETCD Pod Logs", func() { etcdPodLogs() }},
		{"Kube-Controller-Manager Logs", func() { kubeControllerManager() }},
	}

	for _, section := range sections {
		printSection(section.title)
		section.fn()
	}
}

func runIssueAnalysis() {
	printHeader("ISSUE IDENTIFICATION & TROUBLESHOOTING")

	sections := []struct {
		title string
		fn    func()
	}{
		{"ClusterVersion Status", func() { clusterversionIssues() }},
		{"Degraded Cluster Operators", func() { degradedOperators() }},
		{"Degraded Operators Detailed Analysis", func() { degradedOperatorsDescription() }},
		{"Degraded Machine Config Pools", func() { degradedMCP() }},
		{"Degraded MCPs Detailed Analysis", func() { degradedMCPDescription() }},
		{"Machines Not in Running State", func() { machinePhase() }},
		{"Degraded Machines Detailed Analysis", func() { degradedMachinesDescription() }},
		{"Degraded Nodes", func() { degradedNodes() }},
		{"Degraded Nodes Detailed Analysis", func() { degradedNodesDescription() }},
		{"Pods Not in Running/Succeeded State", func() { podsNotRunning() }},
		{"Machine-Config-Daemon Logs (Degraded Nodes)", func() { mcdPodLogs() }},
	}

	for _, section := range sections {
		printSection(section.title)
		section.fn()
	}
}

// ============================================================================
// HEALTH ANALYSIS FUNCTIONS
// ============================================================================

func infrastructure() {
	pattern := filepath.Join(cfg.mustGatherPath, "*/cluster-scoped-resources/config.openshift.io/infrastructures.yaml")
	files, _ := filepath.Glob(pattern)

	if len(files) == 0 {
		printWarning("infrastructures.yaml not found")
		printTroubleshoot([]string{
			"Must-gather may be incomplete or corrupted",
			"Re-collect must-gather: oc adm must-gather",
			"Verify cluster-scoped-resources directory exists",
		})
		return
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		printError("Error reading file: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	inRange := false
	for _, line := range lines {
		if strings.Contains(line, "uid:") {
			inRange = true
			continue
		}
		if strings.Contains(line, "kind:") {
			break
		}
		if inRange {
			fmt.Println(line)
		}
	}

	if cfg.verbose {
		printInfo("\nInfrastructure information shows platform type, API endpoints, and cluster topology")
	}
}

func etcdEndpointHealth() {
	pattern := filepath.Join(cfg.mustGatherPath, "*/etcd_info/endpoint_health.json")
	files, _ := filepath.Glob(pattern)

	if len(files) == 0 {
		printWarning("endpoint_health.json not found")
		printTroubleshoot([]string{
			"ETCD diagnostics may not have been collected",
			"This is critical for cluster health assessment",
			"Ensure must-gather includes ETCD information",
		})
		return
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		printError("Error reading file: %v", err)
		return
	}

	var health []ETCDEndpointHealth
	if err := json.Unmarshal(content, &health); err != nil {
		printError("Error parsing JSON: %v", err)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENDPOINT\tHEALTH\tTOOK")
	fmt.Fprintln(w, strings.Repeat("-", 60))

	allHealthy := true
	for _, h := range health {
		status := checkMark
		if !h.Health {
			status = crossMark
			allHealthy = false
		}
		fmt.Fprintf(w, "%s\t%s %v\t%s\n", h.Endpoint, status, h.Health, h.Took)
	}
	w.Flush()

	if !allHealthy {
		printWarning("\nSome ETCD endpoints are unhealthy!")
		printTroubleshoot([]string{
			"Check ETCD pod logs for errors",
			"Verify network connectivity between ETCD members",
			"Check master node resources (CPU, memory, disk)",
			"Review ETCD certificates and authentication",
			"Consult: https://docs.openshift.com/container-platform/latest/backup_and_restore/control_plane_backup_and_restore/disaster_recovery/about-disaster-recovery.html",
		})
	} else if cfg.verbose {
		printSuccess("\nAll ETCD endpoints are healthy")
	}
}

func etcdEndpointStatus() {
	pattern := filepath.Join(cfg.mustGatherPath, "*/etcd_info/endpoint_status.json")
	files, _ := filepath.Glob(pattern)

	if len(files) == 0 {
		printWarning("endpoint_status.json not found")
		return
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		printError("Error reading file: %v", err)
		return
	}

	var statuses []ETCDEndpointStatus
	if err := json.Unmarshal(content, &statuses); err != nil {
		printError("Error parsing JSON: %v", err)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENDPOINT\tMEMBER-ID\tLEADER-ID\tVERSION\tDB-SIZE(MB)\tRAFT-TERM\tRAFT-INDEX\tRAFT-APPLIED")
	fmt.Fprintln(w, strings.Repeat("-", 120))

	var leaderID int64
	dbSizes := make(map[int64]int64)
	for _, s := range statuses {
		dbSizeMB := float64(s.Status.DBSize) / 1024 / 1024
		leaderID = s.Status.Leader
		dbSizes[s.Status.Header.MemberID] = s.Status.DBSize

		fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%.2f\t%d\t%d\t%d\n",
			s.Endpoint,
			s.Status.Header.MemberID,
			s.Status.Leader,
			s.Status.Version,
			dbSizeMB,
			s.Status.Header.RaftTerm,
			s.Status.RaftIndex,
			s.Status.RaftAppliedIndex,
		)
	}
	w.Flush()

	if cfg.verbose {
		printInfo("\nETCD Status Analysis:")
		fmt.Printf("  • Leader ID: %d\n", leaderID)
		fmt.Printf("  • Total members: %d\n", len(statuses))

		maxDBSize := int64(0)
		for _, size := range dbSizes {
			if size > maxDBSize {
				maxDBSize = size
			}
		}
		maxDBSizeMB := float64(maxDBSize) / 1024 / 1024
		if maxDBSizeMB > 8000 {
			printWarning(fmt.Sprintf("  • Large ETCD database detected (%.2f MB)", maxDBSizeMB))
			printTroubleshoot([]string{
				"Consider ETCD defragmentation if DB size > 8GB",
				"Review object counts and resource quotas",
				"Check for excessive events or log entries",
			})
		}
	}
}

func etcdMemberList() {
	pattern := filepath.Join(cfg.mustGatherPath, "*/etcd_info/member_list.json")
	files, _ := filepath.Glob(pattern)

	if len(files) == 0 {
		printWarning("member_list.json not found")
		return
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		printError("Error reading file: %v", err)
		return
	}

	var memberList ETCDMemberList
	if err := json.Unmarshal(content, &memberList); err != nil {
		printError("Error parsing JSON: %v", err)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPEER-ADDRS\tCLIENT-ADDRS")
	fmt.Fprintln(w, strings.Repeat("-", 100))
	for _, m := range memberList.Members {
		peerAddrs := strings.Join(m.PeerURLs, ", ")
		clientAddrs := strings.Join(m.ClientURLs, ", ")
		fmt.Fprintf(w, "%s\t%s\t%s\n", m.Name, peerAddrs, clientAddrs)
	}
	w.Flush()

	if cfg.verbose && len(memberList.Members) != 3 {
		printWarning(fmt.Sprintf("\nNon-standard ETCD member count: %d", len(memberList.Members)))
		printInfo("Recommended: 3 or 5 members for HA clusters")
	}
}

func clusterversion() {
	runOMGCommand("get", "clusterversion")

	pattern := filepath.Join(cfg.mustGatherPath, "*/cluster-scoped-resources/config.openshift.io/clusterversions.yaml")
	files, _ := filepath.Glob(pattern)

	if len(files) == 0 {
		printWarning("clusterversions.yaml not found")
		return
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		printError("Error reading file: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")

	printSubSection("ClusterVersion Spec")
	printSectionBetween(lines, "uid:", "version:", false, true)

	printSubSection("ClusterVersion Conditions")
	printSectionBetween(lines, "lastTransitionTime:", "desired:", false, false)

	printSubSection("ClusterVersion History")
	printSectionBetween(lines, "completionTime:", "observedGeneration:", false, false)
}

func installConfigYAML() {
	pattern := filepath.Join(cfg.mustGatherPath, "*/namespaces/kube-system/core/configmaps.yaml")
	files, _ := filepath.Glob(pattern)

	if len(files) == 0 {
		printWarning("install-config.yaml not found")
		printTroubleshoot([]string{
			"Install config may have been removed post-installation",
			"This is normal for some clusters",
			"Check cluster documentation for deployment details",
		})
		return
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		printError("Error reading file: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	inSection := false
	foundConfig := false
	for _, line := range lines {
		if strings.Contains(line, "install-config") {
			inSection = true
			foundConfig = true
		}
		if inSection {
			if strings.Contains(line, "kind:") && !strings.Contains(line, "install-config") {
				break
			}
			fmt.Println(line)
		}
	}

	if !foundConfig && cfg.verbose {
		printInfo("Install configuration shows deployment topology, networking, and platform details")
	}
}

func clusterWideProxy() {
	pattern := filepath.Join(cfg.mustGatherPath, "*/cluster-scoped-resources/config.openshift.io/proxies/cluster.yaml")
	files, _ := filepath.Glob(pattern)

	if len(files) == 0 {
		printWarning("cluster-wide proxy details not found")
		if cfg.verbose {
			printInfo("No proxy configuration - cluster may not require proxy")
		}
		return
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		printError("Error reading file: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	inRange := false
	hasProxy := false
	for _, line := range lines {
		if strings.Contains(line, "uid:") {
			inRange = true
			continue
		}
		if strings.Contains(line, "kind:") {
			break
		}
		if inRange {
			fmt.Println(line)
			if strings.Contains(line, "httpProxy:") || strings.Contains(line, "httpsProxy:") {
				hasProxy = true
			}
		}
	}

	if hasProxy && cfg.verbose {
		printInfo("\nProxy configuration detected - ensure no-proxy settings include cluster networks")
	}
}

func clusterOperator() {
	runOMGCommand("get", "co")
}

func nodes() {
	runOMGCommand("get", "nodes", "-o", "wide")
}

func machineconfiguration() {
	pattern := filepath.Join(cfg.mustGatherPath, "*/cluster-scoped-resources/core/nodes/*.yaml")
	files, _ := filepath.Glob(pattern)

	if len(files) == 0 {
		printWarning("No node configurations found")
		return
	}

	mismatchFound := false
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		var hostname, current, desired string

		for _, line := range lines {
			if strings.Contains(line, "kubernetes.io/hostname") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					hostname = strings.TrimSpace(parts[1])
				}
			}
			if strings.Contains(line, "machineconfiguration.openshift.io/currentConfig") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					current = strings.TrimSpace(parts[1])
				}
			}
			if strings.Contains(line, "machineconfiguration.openshift.io/desiredConfig") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					desired = strings.TrimSpace(parts[1])
				}
			}
		}

		if hostname != "" {
			status := checkMark
			if current != desired {
				status = warningMark
				mismatchFound = true
			}
			fmt.Printf("%s %s\n", status, hostname)
			fmt.Printf("  Current:  %s\n", current)
			fmt.Printf("  Desired:  %s\n\n", desired)
		}
	}

	if mismatchFound {
		printWarning("Configuration drift detected!")
		printTroubleshoot([]string{
			"Some nodes have current config != desired config",
			"Check MachineConfigPool status",
			"Review machine-config-daemon logs",
			"Node may be updating or stuck in update",
		})
	}
}

func mcp() {
	runOMGCommand("get", "mcp")
}

func machine() {
	runOMGCommand("get", "machine", "-n", "openshift-machine-api")
}

func machineset() {
	runOMGCommand("get", "machineset", "-n", "openshift-machine-api")
}

func pods() {
	cmd := exec.Command("omg", "get", "pod", "-o", "wide", "-A")
	output, err := cmd.Output()
	if err != nil {
		printError("Error running omg command: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	failingCount := 0
	for _, line := range lines {
		if !strings.Contains(line, "Running") && !strings.Contains(line, "Succeeded") && line != "" {
			fmt.Println(line)
			failingCount++
		}
	}

	if failingCount > 1 && cfg.verbose {
		printWarning(fmt.Sprintf("\nFound %d failing pods", failingCount-1))
		printInfo("Review pod logs and events for root cause analysis")
	}
}

func podRestart() {
	cmd := exec.Command("omg", "get", "pod", "-A")
	output, err := cmd.Output()
	if err != nil {
		printError("Error running omg command: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	highRestartCount := 0
	for i, line := range lines {
		if i == 0 {
			fmt.Println(line)
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			restarts, err := strconv.Atoi(fields[4])
			if err == nil && restarts > 10 {
				fmt.Println(line)
				highRestartCount++
			}
		}
	}

	if highRestartCount > 0 {
		printWarning(fmt.Sprintf("\n%d pods with excessive restarts detected", highRestartCount))
		printTroubleshoot([]string{
			"High restart count indicates instability",
			"Check pod logs for crash reasons",
			"Review resource limits and requests",
			"Check for liveness/readiness probe failures",
			"Investigate OOMKilled events",
		})
	}
}

func kubeApiserver() {
	masterNodes := getMasterNodes("kube-apiserver-")
	if len(masterNodes) == 0 {
		printWarning("No master nodes found for kube-apiserver")
		return
	}

	for _, node := range masterNodes {
		logPath := filepath.Join(cfg.mustGatherPath, "*/namespaces/openshift-kube-apiserver/pods", node, "kube-apiserver/kube-apiserver/logs/current.log")
		files, _ := filepath.Glob(logPath)

		if len(files) == 0 {
			printWarning(fmt.Sprintf("%s pod logs not found", node))
			continue
		}

		printSubSection(node)
		printTailLines(files[0], 10)
	}
}

func etcdPodLogs() {
	masterNodes := getMasterNodes("etcd-")
	if len(masterNodes) == 0 {
		printWarning("No master nodes found for ETCD")
		return
	}

	for _, node := range masterNodes {
		logPath := filepath.Join(cfg.mustGatherPath, "*/namespaces/openshift-etcd/pods", node, "etcd/etcd/logs/current.log")
		files, _ := filepath.Glob(logPath)

		if len(files) == 0 {
			printWarning(fmt.Sprintf("%s pod logs not found", node))
			continue
		}

		printSubSection(node)
		printTailLines(files[0], 10)
	}
}

func kubeControllerManager() {
	masterNodes := getMasterNodes("kube-controller-manager-")
	if len(masterNodes) == 0 {
		printWarning("No master nodes found for kube-controller-manager")
		return
	}

	for _, node := range masterNodes {
		logPath := filepath.Join(cfg.mustGatherPath, "*/namespaces/openshift-kube-controller-manager/pods", node, "kube-controller-manager/kube-controller-manager/logs/current.log")
		files, _ := filepath.Glob(logPath)

		if len(files) == 0 {
			printWarning(fmt.Sprintf("%s pod logs not found", node))
			continue
		}

		printSubSection(node)
		printTailLines(files[0], 10)
	}
}

// ============================================================================
// ISSUE ANALYSIS FUNCTIONS
// ============================================================================

func clusterversionIssues() {
	runOMGCommand("get", "clusterversion")
}

func degradedOperators() {
	cmd := exec.Command("omg", "get", "co")
	output, err := cmd.Output()
	if err != nil {
		printError("Error running omg command: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	degradedCount := 0
	healthyOperators := true

	for i, line := range lines {
		if i == 0 {
			fmt.Println(line)
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			available := fields[2]
			progressing := fields[3]
			degraded := fields[4]

			if available != "True" || progressing != "False" || degraded != "False" {
				fmt.Println(line)
				degradedCount++
				healthyOperators = false
			}
		}
	}

	if healthyOperators {
		printSuccess("\nAll cluster operators are working fine!")
	} else {
		printError(fmt.Sprintf("\n%d degraded operators found!", degradedCount))
		printTroubleshoot([]string{
			"Check operator logs: oc logs -n <namespace> <pod>",
			"Review operator conditions in detailed section below",
			"Verify all prerequisites are met (network, storage, etc.)",
			"Check for resource constraints",
			"Review recent cluster changes or updates",
		})
	}
}

func degradedOperatorsDescription() {
	cmd := exec.Command("omg", "get", "co")
	output, err := cmd.Output()
	if err != nil {
		printError("Error running omg command: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	var degradedOps []string

	for i, line := range lines {
		if i == 0 {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			available := fields[2]
			progressing := fields[3]
			degraded := fields[4]

			if available != "True" || progressing != "False" || degraded != "False" {
				degradedOps = append(degradedOps, fields[0])
			}
		}
	}

	if len(degradedOps) == 0 {
		printSuccess("Not required - all operators are available!")
		return
	}

	for _, op := range degradedOps {
		printSubSection(fmt.Sprintf("%s operator description", op))
		runOMGCommand("get", "co", op, "-o", "yaml")
		fmt.Println()
	}
}

func degradedMCP() {
	cmd := exec.Command("omg", "get", "mcp")
	output, err := cmd.Output()
	if err != nil {
		printError("Error running omg command: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	degradedCount := 0
	healthyMCPs := true

	for i, line := range lines {
		if i == 0 {
			fmt.Println(line)
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			updated := fields[2]
			updating := fields[3]
			degraded := fields[4]

			if updated != "True" || updating != "False" || degraded != "False" {
				fmt.Println(line)
				degradedCount++
				healthyMCPs = false
			}
		}
	}

	if healthyMCPs {
		printSuccess("\nNo machine-config-pool degraded!")
	} else {
		printError(fmt.Sprintf("\n%d degraded machine-config-pools found!", degradedCount))
		printTroubleshoot([]string{
			"Check MCP conditions for specific errors",
			"Review machine-config-daemon logs on affected nodes",
			"Verify disk space on nodes",
			"Check for file system corruption",
			"Review recent MachineConfig changes",
		})
	}
}

func degradedMCPDescription() {
	cmd := exec.Command("omg", "get", "mcp")
	output, err := cmd.Output()
	if err != nil {
		printError("Error running omg command: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	var degradedMCPs []string

	for i, line := range lines {
		if i == 0 {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {
			updated := fields[2]
			updating := fields[3]
			degraded := fields[4]

			if updated != "True" || updating != "False" || degraded != "False" {
				degradedMCPs = append(degradedMCPs, fields[0])
			}
		}
	}

	if len(degradedMCPs) == 0 {
		printSuccess("Not required - all machine-config-pools are available!")
		return
	}

	for _, mcp := range degradedMCPs {
		printSubSection(fmt.Sprintf("%s machine-config-pool description", mcp))
		runOMGCommand("get", "mcp", mcp, "-o", "yaml")
		fmt.Println()
	}
}

func machinePhase() {
	cmd := exec.Command("omg", "get", "machine", "-n", "openshift-machine-api")
	output, err := cmd.Output()
	if err != nil {
		if cfg.verbose {
			printInfo("Machine API not available - may be BareMetal UPI cluster")
		}
		return
	}

	lines := strings.Split(string(output), "\n")
	degradedCount := 0
	allRunning := true

	for i, line := range lines {
		if i == 0 {
			fmt.Println(line)
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] != "Running" {
			fmt.Println(line)
			degradedCount++
			allRunning = false
		}
	}

	if allRunning {
		printSuccess("\nAll machines are Running fine or BareMetal UPI cluster (no machine-api)")
	} else {
		printError(fmt.Sprintf("\n%d machines not in Running state!", degradedCount))
		printTroubleshoot([]string{
			"Check machine controller logs",
			"Verify cloud provider credentials",
			"Check quota limits in cloud provider",
			"Review machine events for provisioning errors",
			"Verify network connectivity to cloud API",
		})
	}
}

func degradedMachinesDescription() {
	cmd := exec.Command("omg", "get", "machine", "-n", "openshift-machine-api")
	output, err := cmd.Output()
	if err != nil {
		printSuccess("Not required - BareMetal UPI cluster or machine-api not available")
		return
	}

	lines := strings.Split(string(output), "\n")
	var degradedMachines []string

	for i, line := range lines {
		if i == 0 {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] != "Running" {
			degradedMachines = append(degradedMachines, fields[0])
		}
	}

	if len(degradedMachines) == 0 {
		printSuccess("Not required - all machines are Running or BareMetal UPI cluster")
		return
	}

	for _, machine := range degradedMachines {
		printSubSection(fmt.Sprintf("%s machine description", machine))
		runOMGCommand("get", "machine", machine, "-n", "openshift-machine-api", "-o", "yaml")
		fmt.Println()
	}
}

func degradedNodes() {
	cmd := exec.Command("omg", "get", "nodes", "-o", "wide")
	output, err := cmd.Output()
	if err != nil {
		printError("Error running omg command: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	degradedCount := 0
	allReady := true

	for i, line := range lines {
		if i == 0 {
			fmt.Println(line)
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] != "Ready" {
			fmt.Println(line)
			degradedCount++
			allReady = false
		}
	}

	if allReady {
		printSuccess("\nAll nodes are in Ready state!")
	} else {
		printError(fmt.Sprintf("\n%d nodes not in Ready state!", degradedCount))
		printTroubleshoot([]string{
			"SSH to node and check system logs: journalctl -xe",
			"Check kubelet status: systemctl status kubelet",
			"Verify node resources: df -h, free -m",
			"Check network connectivity",
			"Review node conditions in detailed section below",
			"Check for disk pressure, memory pressure, or PID pressure",
		})
	}
}

func degradedNodesDescription() {
	cmd := exec.Command("omg", "get", "nodes")
	output, err := cmd.Output()
	if err != nil {
		printError("Error running omg command: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	var degradedNodes []string

	for i, line := range lines {
		if i == 0 {
			continue
		}

		if strings.Contains(line, "NotReady") || strings.Contains(line, "SchedulingDisabled") {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				degradedNodes = append(degradedNodes, fields[0])
			}
		}
	}

	if len(degradedNodes) == 0 {
		printSuccess("Not required - all nodes are in Ready state!")
		return
	}

	for _, node := range degradedNodes {
		pattern := filepath.Join(cfg.mustGatherPath, "*/cluster-scoped-resources/core/nodes", node+".yaml")
		files, _ := filepath.Glob(pattern)

		if len(files) == 0 {
			printWarning(fmt.Sprintf("%s node description not found", node))
			continue
		}

		printSubSection(node)
		content, err := os.ReadFile(files[0])
		if err != nil {
			printError("Error reading file: %v", err)
			continue
		}

		lines := strings.Split(string(content), "\n")

		printInfo("Node Metadata:")
		printSectionBetween(lines, "apiVersion", "daemonEndpoints", false, false)

		printInfo("\nNode Status:")
		printSectionBetween(lines, "nodeInfo", "", true, false)
	}
}

func podsNotRunning() {
	cmd := exec.Command("omg", "get", "pod", "-o", "wide", "-A")
	output, err := cmd.Output()
	if err != nil {
		printError("Error running omg command: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	failingCount := 0
	allRunning := true

	for i, line := range lines {
		if i == 0 {
			fmt.Println(line)
			continue
		}

		if !strings.Contains(line, "Running") && !strings.Contains(line, "Succeeded") && line != "" {
			fmt.Println(line)
			failingCount++
			allRunning = false
		}
	}

	if allRunning {
		printSuccess("\nAll pods are in Running state!")
	} else {
		printError(fmt.Sprintf("\n%d pods not in Running/Succeeded state!", failingCount))
		printTroubleshoot([]string{
			"Check pod logs: oc logs <pod> -n <namespace>",
			"Check pod events: oc describe pod <pod> -n <namespace>",
			"Common causes:",
			"  - ImagePullBackOff: Check image registry access",
			"  - CrashLoopBackOff: Application error, check logs",
			"  - Pending: Resource constraints or scheduling issues",
			"  - Error/Failed: Check pod logs and events",
			"  - OOMKilled: Increase memory limits",
		})
	}
}

func mcdPodLogs() {
	cmd := exec.Command("omg", "get", "nodes")
	output, err := cmd.Output()
	if err != nil {
		printError("Error running omg command: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	var degradedNodes []string

	for i, line := range lines {
		if i == 0 {
			continue
		}

		if strings.Contains(line, "NotReady") || strings.Contains(line, "SchedulingDisabled") {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				degradedNodes = append(degradedNodes, fields[0])
			}
		}
	}

	if len(degradedNodes) == 0 {
		printSuccess("Not required - all nodes are in Ready state!")
		return
	}

	for _, node := range degradedNodes {
		cmd := exec.Command("omg", "get", "pod", "-o", "wide", "-n", "openshift-machine-config-operator")
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		lines := strings.Split(string(output), "\n")
		var mcdPod string

		for _, line := range lines {
			if strings.Contains(line, node) && strings.Contains(line, "machine-config-daemon") {
				fields := strings.Fields(line)
				if len(fields) >= 1 {
					mcdPod = fields[0]
					break
				}
			}
		}

		if mcdPod == "" {
			printWarning(fmt.Sprintf("MCD pod not found for node %s", node))
			continue
		}

		logPath := filepath.Join(cfg.mustGatherPath, "*/namespaces/openshift-machine-config-operator/pods", mcdPod, "machine-config-daemon/machine-config-daemon/logs/current.log")
		files, _ := filepath.Glob(logPath)

		if len(files) == 0 {
			printWarning(fmt.Sprintf("%s pod logs not found for degraded %s node", mcdPod, node))
			continue
		}

		content, _ := os.ReadFile(files[0])
		if len(content) == 0 {
			printWarning(fmt.Sprintf("%s pod logs empty for degraded %s node", mcdPod, node))
			continue
		}

		printSubSection(fmt.Sprintf("%s pod logs for degraded %s node", mcdPod, node))
		printTailLines(files[0], 15)
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func getMasterNodes(prefix string) []string {
	pattern := filepath.Join(cfg.mustGatherPath, "*/cluster-scoped-resources/core/nodes/*.yaml")
	files, _ := filepath.Glob(pattern)

	var nodes []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if strings.Contains(string(content), "node-role.kubernetes.io/master: \"\"") {
			lines := strings.Split(string(content), "\n")
			for i, line := range lines {
				if strings.Contains(line, "node-role.kubernetes.io/master: \"\"") {
					for j := i + 1; j < len(lines) && j < i+200; j++ {
						if strings.Contains(lines[j], "resourceVersion:") && j > 0 {
							prevLine := lines[j-1]
							fields := strings.Fields(prevLine)
							if len(fields) >= 2 {
								hostname := fields[1]
								nodes = append(nodes, prefix+hostname)
							}
							break
						}
					}
					break
				}
			}
		}
	}
	return nodes
}

func printTailLines(filename string, n int) {
	file, err := os.Open(filename)
	if err != nil {
		printError("Error opening file: %v", err)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		printError("Error reading file: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	start := len(lines) - n - 1
	if start < 0 {
		start = 0
	}

	for i := start; i < len(lines); i++ {
		if lines[i] != "" {
			fmt.Println(lines[i])
		}
	}
}

func printSectionBetween(lines []string, startMarker, endMarker string, includeStart, includeEnd bool) {
	inSection := false
	for _, line := range lines {
		if strings.Contains(line, startMarker) {
			inSection = true
			if includeStart {
				fmt.Println(line)
			}
			if endMarker == "" {
				continue
			}
			if !includeStart {
				continue
			}
		}
		if endMarker != "" && strings.Contains(line, endMarker) {
			if includeEnd {
				fmt.Println(line)
			}
			break
		}
		if inSection {
			fmt.Println(line)
		}
	}
}

func runOMGCommand(args ...string) {
	cmd := exec.Command("omg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil && cfg.verbose {
		printError("Error running omg command: %v", err)
	}
}

func printHeader(text string) {
	fmt.Printf("\n%s%s╔%s╗%s\n", bold, colorCyan, strings.Repeat("═", len(text)+2), regular)
	fmt.Printf("%s%s║ %s ║%s\n", bold, colorCyan, text, regular)
	fmt.Printf("%s%s╚%s╝%s\n\n", bold, colorCyan, strings.Repeat("═", len(text)+2), regular)
}

func printSection(text string) {
	fmt.Printf("\n%s%s▶ %s%s\n", bold, colorYellow, text, regular)
	fmt.Println(strings.Repeat("─", 80))
}

func printSubSection(text string) {
	fmt.Printf("\n%s%s● %s%s\n", colorPurple, bold, text, regular)
}

func printSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s %s%s\n", colorGreen, checkMark, msg, colorReset)
}

func printWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s %s%s\n", colorYellow, warningMark, msg, colorReset)
}

func printError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s %s%s\n", colorRed, crossMark, msg, colorReset)
}

func printInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s %s%s\n", colorCyan, infoMark, msg, colorReset)
}

func printColored(color, text string) {
	if cfg.noColor {
		fmt.Print(text)
	} else {
		fmt.Printf("%s%s%s", color, text, colorReset)
	}
}

func printTroubleshoot(steps []string) {
	if !cfg.verbose {
		return
	}
	fmt.Printf("\n%s%s Troubleshooting Steps:%s\n", colorYellow, warningMark, colorReset)
	for _, step := range steps {
		fmt.Printf("  %s\n", step)
	}
}

func printSummary() {
	fmt.Printf("\n\n%s%s", bold, colorCyan)
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     ANALYSIS COMPLETE                          ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Printf("%s\n", regular)

	fmt.Println("\nNext Steps:")
	fmt.Println("  1. Review all sections marked with warnings or errors")
	fmt.Println("  2. Check detailed descriptions for degraded components")
	fmt.Println("  3. Review pod and component logs for specific errors")
	fmt.Println("  4. Consult OpenShift documentation for specific issues")
	fmt.Println("  5. For production issues, open a Red Hat support case")

	fmt.Printf("\n%sRe-run with -verbose flag for detailed troubleshooting guidance%s\n", colorYellow, colorReset)
	fmt.Printf("%sRe-run with -mode issues to focus on problem identification%s\n\n", colorYellow, colorReset)
}
