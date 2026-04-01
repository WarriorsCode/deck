package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/warriorscode/deck/config"
	"github.com/warriorscode/deck/engine"
	"github.com/warriorscode/deck/status"
)

var configFile string

func main() {
	root := &cobra.Command{
		Use:   "deck",
		Short: "Lightweight local dev orchestrator",
	}

	root.PersistentFlags().StringVarP(&configFile, "file", "f", "deck.yaml", "config file path")

	root.AddCommand(
		upCmd(),
		startCmd(),
		stopCmd(),
		restartCmd(),
		statusCmd(),
		logsCmd(),
		initCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadConfig() (*config.Config, error) {
	if configFile != "deck.yaml" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("no %s found. Run 'deck init' to create one", filepath.Base(configFile))
		}
		return config.Parse(data)
	}
	return config.LoadFile(configFile)
}

func newEngine(cfg *config.Config) *engine.Engine {
	dir, _ := os.Getwd()
	deckDir := filepath.Join(dir, ".deck")
	return engine.New(cfg, dir, deckDir)
}

func upCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Start services in foreground with log tailing",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			eng := newEngine(cfg)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := eng.Preflight(ctx); err != nil {
				return err
			}
			if err := eng.Start(); err != nil {
				return err
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
				eng.Stop()
			}()

			engine.TailLogs(ctx, eng.LogConfigs(), os.Stdout)
			return nil
		},
	}
}

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start services in background (detached)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			eng := newEngine(cfg)

			if err := eng.Preflight(context.Background()); err != nil {
				return err
			}
			if err := eng.Start(); err != nil {
				return err
			}

			statuses := eng.Status()
			out, _ := status.Format(statuses, "")
			fmt.Println(out)
			return nil
		},
	}
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop all running services",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			eng := newEngine(cfg)
			eng.Stop()
			fmt.Println("All services stopped.")
			return nil
		},
	}
}

func restartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart all services",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			eng := newEngine(cfg)
			eng.Stop()

			if err := eng.Preflight(context.Background()); err != nil {
				return err
			}
			return eng.Start()
		},
	}
}

func statusCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of services",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			eng := newEngine(cfg)

			statuses := eng.Status()

			for name, dep := range cfg.Deps {
				s := "stopped"
				if engine.CheckShell(context.Background(), ".", dep.Check) {
					s = "running"
				}
				statuses = append(statuses, engine.ServiceStatus{Name: name, Status: s, Type: "dep"})
			}

			out, err := status.Format(statuses, format)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "", "output format: table (default), json, or Go template")
	return cmd
}

func logsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Tail service logs with colored prefixes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			eng := newEngine(cfg)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			engine.TailLogs(ctx, eng.LogConfigs(), os.Stdout)
			return nil
		},
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize deck.yaml and .gitignore",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat("deck.yaml"); os.IsNotExist(err) {
				scaffold := `# deck.yaml — local dev stack configuration
# See: https://github.com/warriorscode/deck

name: myproject

# One-time setup tasks. Only run if check fails.
# bootstrap:
#   - name: Install deps
#     check: test -d node_modules
#     run: npm install

# External dependencies.
# deps:
#   postgres:
#     check: pg_isready -h 127.0.0.1
#     start:
#       - docker run -d --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16
#     stop:
#       - docker stop postgres && docker rm postgres

# Lifecycle hooks.
# hooks:
#   pre-start:
#     - name: Run migrations
#       run: migrate up
#   post-stop: []

# Services to manage.
services:
  app:
    run: echo "replace with your start command"
    # dir: ./src
    # port: 3000
    # color: cyan
`
				if err := os.WriteFile("deck.yaml", []byte(scaffold), 0644); err != nil {
					return err
				}
				fmt.Println("Created deck.yaml")
			} else {
				fmt.Println("deck.yaml already exists, skipping")
			}

			entries := []string{".deck/", "deck.local.yaml"}
			gitignore := ""
			if data, err := os.ReadFile(".gitignore"); err == nil {
				gitignore = string(data)
			}
			var toAdd []string
			for _, entry := range entries {
				if !containsLine(gitignore, entry) {
					toAdd = append(toAdd, entry)
				}
			}
			if len(toAdd) > 0 {
				f, err := os.OpenFile(".gitignore", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}
				defer f.Close()
				for _, entry := range toAdd {
					fmt.Fprintln(f, entry)
				}
				fmt.Printf("Added %v to .gitignore\n", toAdd)
			}

			return nil
		},
	}
}

func containsLine(content, line string) bool {
	for _, existing := range strings.Split(content, "\n") {
		if strings.TrimRight(existing, "\r") == line {
			return true
		}
	}
	return false
}
