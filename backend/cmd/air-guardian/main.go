package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"mumbai-air-guardian/backend/internal/airdata"
	"mumbai-air-guardian/backend/internal/api"
	"mumbai-air-guardian/backend/internal/config"
	"mumbai-air-guardian/backend/internal/gemma"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	root, err := dataRoot()
	if err != nil {
		return err
	}
	dataset, err := airdata.LoadFromRoot(root)
	if err != nil {
		return fmt.Errorf("load verified air-quality snapshot: %w", err)
	}
	outlookConfig, err := config.LoadOutlookConfig(filepath.Join(root, filepath.FromSlash(config.OutlookConfigRelativePath)))
	if err != nil {
		return err
	}
	handler, err := api.NewHandler(dataset, outlookConfig, []string{"http://localhost:4200"})
	if err != nil {
		return err
	}
	explainer, err := gemma.NewServiceFromEnvironment()
	if err != nil {
		log.Printf("Local Gemma disabled by configuration: %v", err)
		explainer = gemma.NewDisabledService("configuration_invalid")
	}
	if err := handler.SetExplainer(explainer); err != nil {
		return fmt.Errorf("configure explanation service: %w", err)
	}

	server := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      40 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen for API: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(listener)
	}()
	log.Printf("Air Guardian API listening on %s", server.Addr)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve API: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shut down API: %w", err)
	}
	return nil
}

func dataRoot() (string, error) {
	if root := os.Getenv("AIR_GUARDIAN_DATA_ROOT"); root != "" {
		return filepath.Abs(root)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	candidates := []string{workingDirectory, filepath.Dir(workingDirectory)}
	for _, candidate := range candidates {
		configPath := filepath.Join(candidate, filepath.FromSlash(config.SourceConfigRelativePath))
		if _, err := os.Stat(configPath); err == nil {
			return candidate, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect data root %q: %w", candidate, err)
		}
	}
	return "", errors.New("data root not found; set AIR_GUARDIAN_DATA_ROOT to the application root")
}
