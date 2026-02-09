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
	"text/tabwriter"
	"time"

	"github.com/nfedorov/port_server/internal/client"
	"github.com/nfedorov/port_server/internal/config"
	"github.com/nfedorov/port_server/internal/model"
	"github.com/nfedorov/port_server/internal/store"
	"github.com/nfedorov/port_server/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println("portctl " + version.String())
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
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: portctl <command> [flags]

Server lifecycle:
  start     Start the port-server daemon
  stop      Stop the port-server daemon
  restart   Restart the port-server daemon
  status    Show port-server daemon status

Commands:
  allocate  Allocate a port
  release   Release port(s)
  list      List allocations
  check     Check if a port is available
  health    Check server health
  version   Print version and exit`)
}

func cmdAllocate(c *client.Client, args []string) {
	fs := flag.NewFlagSet("allocate", flag.ExitOnError)
	app := fs.String("app", "", "application name (required)")
	instance := fs.String("instance", "", "instance name (required)")
	service := fs.String("service", "", "service name (required)")
	port := fs.Int("port", 0, "specific port to allocate (0 = auto-assign)")
	fs.Parse(args)

	if *app == "" || *instance == "" || *service == "" {
		fmt.Fprintln(os.Stderr, "error: --app, --instance, and --service are required")
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
		fmt.Fprintf(os.Stderr, "error: %s/%s/%s is already allocated on port %d (id=%d)\n",
			alloc.App, alloc.Instance, alloc.Service, alloc.Port, alloc.ID)
		os.Exit(1)
	}
	if err == store.ErrPortTaken {
		fmt.Fprintf(os.Stderr, "error: port %d is already allocated to %s/%s/%s (id=%d)\n",
			alloc.Port, alloc.App, alloc.Instance, alloc.Service, alloc.ID)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("allocated port %d (id=%d) for %s/%s/%s\n",
		alloc.Port, alloc.ID, alloc.App, alloc.Instance, alloc.Service)
}

func cmdRelease(c *client.Client, args []string) {
	fs := flag.NewFlagSet("release", flag.ExitOnError)
	id := fs.Int64("id", 0, "allocation ID to release")
	app := fs.String("app", "", "application name")
	instance := fs.String("instance", "", "instance name")
	service := fs.String("service", "", "service name")
	port := fs.Int("port", 0, "port to release")
	fs.Parse(args)

	if *id != 0 {
		if err := c.ReleaseByID(*id); err == store.ErrNotFound {
			fmt.Fprintf(os.Stderr, "error: allocation %d not found\n", *id)
			os.Exit(1)
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("released allocation %d\n", *id)
		return
	}

	if *app == "" && *port == 0 {
		fmt.Fprintln(os.Stderr, "error: --id, --app, or --port is required")
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
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("released %d allocation(s)\n", n)
}

func cmdList(c *client.Client, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	app := fs.String("app", "", "filter by application")
	instance := fs.String("instance", "", "filter by instance")
	service := fs.String("service", "", "filter by service")
	jsonOut := fs.Bool("json", false, "output as JSON")
	fs.Parse(args)

	allocs, err := c.List(store.Filter{
		App:      *app,
		Instance: *instance,
		Service:  *service,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(allocs)
		return
	}

	if len(allocs) == 0 {
		fmt.Println("no allocations")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tAPP\tINSTANCE\tSERVICE\tPORT\tCREATED")
	for _, a := range allocs {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%s\n",
			a.ID, a.App, a.Instance, a.Service, a.Port,
			a.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	w.Flush()
}

func cmdCheck(c *client.Client, args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	port := fs.Int("port", 0, "port to check (required)")
	fs.Parse(args)

	if *port == 0 {
		fmt.Fprintln(os.Stderr, "error: --port is required")
		fs.Usage()
		os.Exit(1)
	}

	status, err := c.CheckPort(*port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if status.Available {
		fmt.Printf("port %d is available\n", *port)
		os.Exit(0)
	}

	fmt.Printf("port %d is allocated to %s/%s/%s (id=%d)\n",
		*port, status.Holder.App, status.Holder.Instance, status.Holder.Service, status.Holder.ID)
	os.Exit(1)
}

func cmdHealth(c *client.Client) {
	if err := c.Health(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func cmdStart() {
	// Check if already running.
	if pid, ok := readPID(); ok {
		if isProcessAlive(pid) {
			fmt.Fprintf(os.Stderr, "port-server is already running (pid %d)\n", pid)
			os.Exit(1)
		}
		// Stale PID file — clean it up.
		os.Remove(config.DefaultPIDPath())
	}

	// Find the port-server binary next to this executable.
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine executable path: %v\n", err)
		os.Exit(1)
	}
	serverBin := filepath.Join(filepath.Dir(exe), "port-server")
	if _, err := os.Stat(serverBin); err != nil {
		fmt.Fprintf(os.Stderr, "error: port-server binary not found at %s\n", serverBin)
		os.Exit(1)
	}

	// Open log file.
	logPath := config.DefaultLogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot create log directory: %v\n", err)
		os.Exit(1)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot open log file: %v\n", err)
		os.Exit(1)
	}

	// Start detached process.
	cmd := exec.Command(serverBin)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		logFile.Close()
		fmt.Fprintf(os.Stderr, "error: failed to start port-server: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "warning: server started (pid %d) but health check not responding\n", cmd.Process.Pid)
		fmt.Fprintf(os.Stderr, "check logs at %s\n", logPath)
		os.Exit(1)
	}

	fmt.Printf("port-server started (pid %d)\n", cmd.Process.Pid)
}

func cmdStop() {
	pid, ok := readPID()
	if !ok {
		fmt.Fprintln(os.Stderr, "port-server is not running (no PID file)")
		os.Exit(1)
	}

	if !isProcessAlive(pid) {
		os.Remove(config.DefaultPIDPath())
		fmt.Fprintln(os.Stderr, "port-server is not running (stale PID file removed)")
		os.Exit(1)
	}

	// Send SIGTERM.
	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to stop port-server: %v\n", err)
		os.Exit(1)
	}

	// Wait for process to exit.
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if !isProcessAlive(pid) {
			// Clean up PID file if server didn't remove it.
			os.Remove(config.DefaultPIDPath())
			fmt.Printf("port-server stopped (pid %d)\n", pid)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "error: port-server (pid %d) did not stop within 5 seconds\n", pid)
	os.Exit(1)
}

func cmdStatus() {
	pid, ok := readPID()
	if !ok {
		fmt.Println("port-server is not running")
		return
	}

	if !isProcessAlive(pid) {
		os.Remove(config.DefaultPIDPath())
		fmt.Println("port-server is not running (stale PID file removed)")
		return
	}

	// Check health endpoint.
	addr := fmt.Sprintf("127.0.0.1:%d", config.DefaultServerPort)
	healthURL := "http://" + addr + "/healthz"
	resp, err := http.Get(healthURL)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			fmt.Printf("port-server is running (pid %d, healthy)\n", pid)
			return
		}
	}

	fmt.Printf("port-server is running (pid %d, not healthy)\n", pid)
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
