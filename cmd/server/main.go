package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nfedorov/port_server/internal/config"
	"github.com/nfedorov/port_server/internal/handler"
	"github.com/nfedorov/port_server/internal/store"
	"github.com/nfedorov/port_server/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	port := flag.Int("port", config.DefaultServerPort, "server listen port")
	dbPath := flag.String("db", config.DefaultDBPath(), "SQLite database path")
	pidFile := flag.String("pidfile", config.DefaultPIDPath(), "PID file path")
	flag.Parse()

	if *showVersion {
		fmt.Println("port-server " + version.String())
		return
	}

	// Ensure DB directory exists.
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
		log.Fatalf("failed to create db directory: %v", err)
	}

	s, err := store.NewSQLite(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer s.Close()

	h := handler.New(s)
	srv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", *port),
		Handler: h.Routes(),
	}

	// Graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", srv.Addr, err)
	}

	// Write PID file.
	if err := os.MkdirAll(filepath.Dir(*pidFile), 0755); err != nil {
		log.Fatalf("failed to create pid directory: %v", err)
	}
	if err := os.WriteFile(*pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		log.Fatalf("failed to write pid file: %v", err)
	}
	defer os.Remove(*pidFile)

	go func() {
		log.Printf("port-server listening on %s", srv.Addr)
		if err := srv.Serve(ln); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}
