package main

import (
	"os"
	"os/signal"
	"syscall"

	"backer/internal/backer"
	"backer/internal/log"
)

// setupSignalHandler sets up signal handling for immediate shutdown.
// SIGTERM is commonly sent by systemd, k8s, and other container orchestrators.
// SIGINT is sent on Ctrl+C.
func setupSignalHandler(server *backer.ServerWrapper) {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalCh
		log.Warnf("Shutdown signal received, shutting down immediately...")
		// Close the embedded http.Server directly.
		if closeErr := server.Close(); closeErr != nil {
			log.Errorf("Error during server shutdown: %v", closeErr)
		}
	}()
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
