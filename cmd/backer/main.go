// Package main implements minimal wrapper to run backer as go application.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"backer/internal/backer"
	"backer/internal/log"
)

var (
	// Version is set at build time via ldflags.
	version = "dev"
)

func main() {
	configPath := flag.String("c", "", "path to config file (default: OS-specific location)")
	showVersion := flag.Bool("version", false, "show version and exit")

	flag.Parse()

	if *showVersion {
		fmt.Printf("backer %s\n", version)

		os.Exit(0)
	}

	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = defaultConfigPath()
	}

	server, err := backer.NewServer(cfgPath)

	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Infof("Starting backer version %s at %s", version, server.Addr)

	if backer.C.NoHTTPS {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	} else {
		if _, err := os.Stat(backer.C.Cert); err != nil {
			log.Fatalf("Certificate file error: %v", err)
		}

		if _, err := os.Stat(backer.C.Key); err != nil {
			log.Fatalf("Certificate key file error: %v", err)
		}

		if err := server.ListenAndServeTLS(backer.C.Cert, backer.C.Key); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}
}

// defaultConfigPath returns the default config file path based on the OS.
func defaultConfigPath() string {
	switch runtime.GOOS {
	case "freebsd", "netbsd", "openbsd", "dragonfly":
		return "/usr/local/etc/backer.json"
	default: // Linux, macOS, and others.
		return "/etc/backer.json"
	}
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
