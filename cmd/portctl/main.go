package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

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
