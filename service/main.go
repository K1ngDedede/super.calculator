package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const defaultPort = 8080

func main() {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", configuredPort()),
		Handler: NewHandler(),
	}

	serverErrors := make(chan error, 1)
	go func() {
		fmt.Printf("Server running on port %d\n", configuredPort())

		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
		return
	case <-shutdownSignal.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		_ = server.Close()
	}

	if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("server stopped with error: %v", err)
	}
}

func configuredPort() int {
	contents, err := os.ReadFile(".env")
	if err != nil {
		return defaultPort
	}

	portValue := ""
	foundPort := false
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if found && strings.TrimSpace(key) == "PORT" {
			portValue = strings.TrimSpace(value)
			foundPort = true
		}
	}
	if !foundPort || portValue == "" {
		return defaultPort
	}
	for _, digit := range portValue {
		if digit < '0' || digit > '9' {
			return defaultPort
		}
	}

	port, err := strconv.Atoi(portValue)
	if err != nil || port < 1 || port > 65535 {
		return defaultPort
	}
	return port
}
