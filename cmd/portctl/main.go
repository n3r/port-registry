package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nfedorov/port_server/internal/client"
	"github.com/nfedorov/port_server/internal/config"
	"github.com/nfedorov/port_server/internal/model"
	"github.com/nfedorov/port_server/internal/skill"
	"github.com/nfedorov/port_server/internal/store"
	"github.com/nfedorov/port_server/internal/ui"
	"github.com/nfedorov/port_server/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println(ui.Bold("portctl") + " " + ui.Subtle(version.String()))
		return
	case "start":
		cmdStart()
		return
	case "stop":
		cmdStop()
		return
	case "restart":
		cmdStop()
		cmdStart()
		return
	case "status":
		cmdStatus()
		return
	case "skill":
		cmdSkill(os.Args[2:])
		return
	}

	addr := os.Getenv("PORT_SERVER_ADDR")
	if addr == "" {
		addr = fmt.Sprintf("127.0.0.1:%d", config.DefaultServerPort)
	}
	c := client.New(addr)

	switch os.Args[1] {
	case "allocate":
		cmdAllocate(c, os.Args[2:])
	case "release":
		cmdRelease(c, os.Args[2:])
	case "list":
		cmdList(c, os.Args[2:])
	case "check":
		cmdCheck(c, os.Args[2:])
	case "health":
		cmdHealth(c)
	default:
		fmt.Fprintln(os.Stderr, ui.Errorf("unknown command: %s", os.Args[1]))
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, ui.UsageTitle("Usage: portctl <command> [flags]"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, ui.UsageHeader("Server lifecycle:"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("start", "Start the port-server daemon"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("stop", "Stop the port-server daemon"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("restart", "Restart the port-server daemon"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("status", "Show port-server daemon status"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, ui.UsageHeader("Commands:"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("allocate", "Allocate a port"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("release", "Release port(s)"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("list", "List allocations"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("check", "Check if a port is available"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("health", "Check server health"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("version", "Print version and exit"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, ui.UsageHeader("Skills:"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("skill", "Manage agent skills"))
}

func detectAppName() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return filepath.Base(strings.TrimSpace(string(out)))
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Base(cwd)
}

func cmdAllocate(c *client.Client, args []string) {
	fs := flag.NewFlagSet("allocate", flag.ExitOnError)
	app := fs.String("app", "", "application name (default: repo or folder name)")
	instance := fs.String("instance", "", "instance name (required)")
	service := fs.String("service", "", "service name (required)")
	port := fs.Int("port", 0, "specific port to allocate (0 = auto-assign)")
	fs.Parse(args)

	if *app == "" {
		*app = detectAppName()
	}
	if *app == "" || *instance == "" || *service == "" {
		fmt.Fprintln(os.Stderr, ui.Error("--app, --instance, and --service are required"))
		fs.Usage()
		os.Exit(1)
	}

	alloc, err := c.Allocate(model.AllocateRequest{
		App:      *app,
		Instance: *instance,
		Service:  *service,
		Port:     *port,
	})
	if err == store.ErrServiceAllocated {
		fmt.Fprintln(os.Stderr, ui.Errorf("%s/%s/%s is already allocated on port %d %s",
			alloc.App, alloc.Instance, alloc.Service, alloc.Port, ui.Subtle(fmt.Sprintf("(id=%d)", alloc.ID))))
		os.Exit(1)
	}
	if err == store.ErrPortTaken {
		fmt.Fprintln(os.Stderr, ui.Errorf("port %d is already allocated to %s/%s/%s %s",
			alloc.Port, alloc.App, alloc.Instance, alloc.Service, ui.Subtle(fmt.Sprintf("(id=%d)", alloc.ID))))
		os.Exit(1)
	}
	if err == store.ErrPortBusy {
		fmt.Fprintln(os.Stderr, ui.Errorf("port %d is in use on the system", *port))
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("%v", err))
		os.Exit(1)
	}

	fmt.Println(ui.Successf("Allocated port %d for %s/%s/%s %s",
		alloc.Port, alloc.App, alloc.Instance, alloc.Service, ui.Subtle(fmt.Sprintf("(id=%d)", alloc.ID))))
}

func cmdRelease(c *client.Client, args []string) {
	fs := flag.NewFlagSet("release", flag.ExitOnError)
	id := fs.Int64("id", 0, "allocation ID to release")
	app := fs.String("app", "", "application name (default: repo or folder name)")
	instance := fs.String("instance", "", "instance name")
	service := fs.String("service", "", "service name")
	port := fs.Int("port", 0, "port to release")
	fs.Parse(args)

	if *app == "" {
		*app = detectAppName()
	}

	if *id != 0 {
		if err := c.ReleaseByID(*id); err == store.ErrNotFound {
			fmt.Fprintln(os.Stderr, ui.Errorf("allocation %d not found", *id))
			os.Exit(1)
		} else if err != nil {
			fmt.Fprintln(os.Stderr, ui.Errorf("%v", err))
			os.Exit(1)
		}
		fmt.Println(ui.Successf("Released allocation %d", *id))
		return
	}

	if *app == "" && *port == 0 {
		fmt.Fprintln(os.Stderr, ui.Error("--id, --app, or --port is required"))
		fs.Usage()
		os.Exit(1)
	}

	n, err := c.ReleaseByFilter(model.ReleaseRequest{
		App:      *app,
		Instance: *instance,
		Service:  *service,
		Port:     *port,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("%v", err))
		os.Exit(1)
	}
	fmt.Println(ui.Successf("Released %d allocation(s)", n))
}

func cmdList(c *client.Client, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	app := fs.String("app", "", "filter by application (default: repo or folder name)")
	instance := fs.String("instance", "", "filter by instance")
	service := fs.String("service", "", "filter by service")
	jsonOut := fs.Bool("json", false, "output as JSON")
	fs.Parse(args)

	if *app == "" {
		*app = detectAppName()
	}

	allocs, err := c.List(store.Filter{
		App:      *app,
		Instance: *instance,
		Service:  *service,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("%v", err))
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(allocs)
		return
	}

	if len(allocs) == 0 {
		fmt.Println(ui.Info("No allocations"))
		return
	}

	rows := make([][]string, len(allocs))
	for i, a := range allocs {
		rows[i] = []string{
			fmt.Sprintf("%d", a.ID),
			a.App,
			a.Instance,
			a.Service,
			fmt.Sprintf("%d", a.Port),
			a.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	fmt.Println(ui.Table(
		[]string{"ID", "APP", "INSTANCE", "SERVICE", "PORT", "CREATED"},
		rows,
	))
}

func cmdCheck(c *client.Client, args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	port := fs.Int("port", 0, "port to check (required)")
	fs.Parse(args)

	if *port == 0 {
		fmt.Fprintln(os.Stderr, ui.Error("--port is required"))
		fs.Usage()
		os.Exit(1)
	}

	status, err := c.CheckPort(*port)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("%v", err))
		os.Exit(1)
	}

	if status.Available {
		fmt.Println(ui.Successf("Port %d is available", *port))
		os.Exit(0)
	}

	fmt.Println(ui.Warningf("Port %d is allocated to %s/%s/%s %s",
		*port, status.Holder.App, status.Holder.Instance, status.Holder.Service,
		ui.Subtle(fmt.Sprintf("(id=%d)", status.Holder.ID))))
	os.Exit(1)
}

func cmdHealth(c *client.Client) {
	if err := c.Health(); err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("%v", err))
		os.Exit(1)
	}
	fmt.Println(ui.Success("Healthy"))
}

func cmdStart() {
	// Check if already running.
	if pid, ok := readPID(); ok {
		if isProcessAlive(pid) {
			fmt.Fprintln(os.Stderr, ui.Warningf("Server is already running %s", ui.Subtle(fmt.Sprintf("(pid %d)", pid))))
			os.Exit(1)
		}
		// Stale PID file — clean it up.
		os.Remove(config.DefaultPIDPath())
	}

	// Find the port-server binary next to this executable.
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("cannot determine executable path: %v", err))
		os.Exit(1)
	}
	serverBin := filepath.Join(filepath.Dir(exe), "port-server")
	if _, err := os.Stat(serverBin); err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("port-server binary not found at %s", serverBin))
		os.Exit(1)
	}

	// Open log file.
	logPath := config.DefaultLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("cannot create log directory: %v", err))
		os.Exit(1)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("cannot open log file: %v", err))
		os.Exit(1)
	}

	// Start detached process.
	cmd := exec.Command(serverBin)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		logFile.Close()
		fmt.Fprintln(os.Stderr, ui.Errorf("failed to start port-server: %v", err))
		os.Exit(1)
	}
	logFile.Close()

	// Wait briefly for the server to become healthy.
	addr := fmt.Sprintf("127.0.0.1:%d", config.DefaultServerPort)
	healthURL := "http://" + addr + "/healthz"
	healthy := false
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err := http.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				healthy = true
				break
			}
		}
	}

	if !healthy {
		fmt.Fprintln(os.Stderr, ui.Warningf("Server started %s but health check not responding",
			ui.Subtle(fmt.Sprintf("(pid %d)", cmd.Process.Pid))))
		fmt.Fprintln(os.Stderr, ui.Infof("Check logs at %s", logPath))
		os.Exit(1)
	}

	fmt.Println(ui.Successf("Server started %s", ui.Subtle(fmt.Sprintf("(pid %d)", cmd.Process.Pid))))
}

func cmdStop() {
	pid, ok := readPID()
	if !ok {
		fmt.Fprintln(os.Stderr, ui.Error("Server is not running (no PID file)"))
		os.Exit(1)
	}

	if !isProcessAlive(pid) {
		os.Remove(config.DefaultPIDPath())
		fmt.Fprintln(os.Stderr, ui.Error("Server is not running (stale PID file removed)"))
		os.Exit(1)
	}

	// Send SIGTERM.
	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("%v", err))
		os.Exit(1)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("failed to stop port-server: %v", err))
		os.Exit(1)
	}

	// Wait for process to exit.
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if !isProcessAlive(pid) {
			// Clean up PID file if server didn't remove it.
			os.Remove(config.DefaultPIDPath())
			fmt.Println(ui.Successf("Server stopped %s", ui.Subtle(fmt.Sprintf("(pid %d)", pid))))
			return
		}
	}

	fmt.Fprintln(os.Stderr, ui.Errorf("port-server %s did not stop within 5 seconds", ui.Subtle(fmt.Sprintf("(pid %d)", pid))))
	os.Exit(1)
}

func cmdStatus() {
	pid, ok := readPID()
	if !ok {
		fmt.Println(ui.Subtle(ui.SymBullet + " Server is not running"))
		return
	}

	if !isProcessAlive(pid) {
		os.Remove(config.DefaultPIDPath())
		fmt.Println(ui.Subtle(ui.SymBullet + " Server is not running (stale PID file removed)"))
		return
	}

	// Check health endpoint.
	addr := fmt.Sprintf("127.0.0.1:%d", config.DefaultServerPort)
	healthURL := "http://" + addr + "/healthz"
	resp, err := http.Get(healthURL)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			fmt.Println(ui.Successf("Server is running %s",
				ui.Subtle(fmt.Sprintf("(pid %d)", pid))+" "+ui.StyleSuccess.Render("healthy")))
			return
		}
	}

	fmt.Println(ui.Warningf("Server is running %s",
		ui.Subtle(fmt.Sprintf("(pid %d)", pid))+" "+ui.StyleWarning.Render("not healthy")))
}

func readPID() (int, bool) {
	data, err := os.ReadFile(config.DefaultPIDPath())
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	return pid, true
}

func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}

func cmdSkill(args []string) {
	if len(args) == 0 {
		skillUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "install":
		cmdSkillInstall(args[1:])
	default:
		fmt.Fprintln(os.Stderr, ui.Errorf("unknown skill command: %s", args[0]))
		skillUsage()
		os.Exit(1)
	}
}

func cmdSkillInstall(args []string) {
	fs := flag.NewFlagSet("skill install", flag.ExitOnError)
	global := fs.Bool("global", false, "install to global platforms (~/.claude, ~/.codex, ~/.agents)")
	fs.Parse(args)

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("cannot determine home directory: %v", err))
		os.Exit(1)
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.Errorf("cannot determine working directory: %v", err))
		os.Exit(1)
	}

	result := skill.Install(home, cwd, *global)

	for _, p := range result.Installed {
		fmt.Println(ui.Successf("Installed to %s %s", p.Name, ui.Subtle(p.Dir)))
	}
	for _, p := range result.Skipped {
		fmt.Println(ui.Infof("Skipped %s %s", p.Name, ui.Subtle("(not detected)")))
	}
	for _, e := range result.Errors {
		fmt.Fprintln(os.Stderr, ui.Errorf("Failed to install to %s: %v", e.Platform.Name, e.Err))
	}

	if len(result.Installed) == 0 && len(result.Errors) == 0 {
		if *global {
			fmt.Println(ui.Warning("No agent platforms detected"))
			fmt.Println(ui.Infof("Create ~/.claude, ~/.codex, or ~/.agents to enable a platform"))
		} else {
			fmt.Println(ui.Warning("No project directory available"))
		}
	}
}

func skillUsage() {
	fmt.Fprintln(os.Stderr, ui.UsageTitle("Usage: portctl skill <command>"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, ui.UsageHeader("Commands:"))
	fmt.Fprintln(os.Stderr, ui.UsageCommand("install", "Install agent skill locally (use --global for global platforms)"))
}
