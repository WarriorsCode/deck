package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	deck "github.com/warriorscode/deck"
	"github.com/warriorscode/deck/config"
	"github.com/warriorscode/deck/engine"
	"github.com/warriorscode/deck/scaffold"
	"github.com/warriorscode/deck/status"
)

var configFile string

func main() {
	root := &cobra.Command{
		Use:     "deck",
		Short:   "Lightweight local dev orchestrator",
		Version: deck.Version,
	}

	root.PersistentFlags().StringVarP(&configFile, "file", "f", "deck.yaml", "config file path")

	root.AddCommand(
		upCmd(),
		startCmd(),
		stopCmd(),
		restartCmd(),
		statusCmd(),
		logsCmd(),
		runCmd(),
		doctorCmd(),
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
		Use:   "up [services...]",
		Short: "Start services in foreground with log tailing",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.ValidateServiceNames(args); err != nil {
				return err
			}
			filter := cfg.ExpandDeps(args)
			eng := newEngine(cfg)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := eng.Preflight(ctx); err != nil {
				return err
			}
			if err := eng.Start(filter); err != nil {
				return err
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer shutCancel()
				// Second signal force-exits.
				go func() {
					<-sigCh
					os.Exit(1)
				}()
				if len(filter) > 0 {
					eng.StopServices(filter)
				} else {
					eng.Shutdown(shutCtx)
				}
				cancel()
			}()

			engine.TailLogs(ctx, eng.LogConfigs(filter), os.Stdout)
			return nil
		},
	}
}

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start [services...]",
		Short: "Start services in background (detached)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.ValidateServiceNames(args); err != nil {
				return err
			}
			filter := cfg.ExpandDeps(args)
			eng := newEngine(cfg)

			if err := eng.Preflight(context.Background()); err != nil {
				return err
			}
			if err := eng.Start(filter); err != nil {
				return err
			}

			statuses := eng.Status(filter)
			out, _ := status.Format(statuses, "")
			fmt.Println(out)
			return nil
		},
	}
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop [services...]",
		Short: "Stop running services",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.ValidateServiceNames(args); err != nil {
				return err
			}
			eng := newEngine(cfg)
			if len(args) > 0 {
				eng.StopServices(args)
				fmt.Printf("Stopped: %s\n", strings.Join(args, ", "))
			} else {
				eng.Stop()
				fmt.Println("All services stopped.")
			}
			return nil
		},
	}
}

func restartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart [services...]",
		Short: "Restart services",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.ValidateServiceNames(args); err != nil {
				return err
			}
			filter := cfg.ExpandDeps(args)
			eng := newEngine(cfg)
			if len(filter) > 0 {
				eng.StopServices(filter)
			} else {
				eng.Stop()
			}

			if err := eng.Preflight(context.Background()); err != nil {
				return err
			}
			return eng.Start(filter)
		},
	}
}

func statusCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "status [services...]",
		Short: "Show status of services",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.ValidateServiceNames(args); err != nil {
				return err
			}
			eng := newEngine(cfg)

			statuses := eng.Status(args)

			if len(args) == 0 {
				baseEnv, _ := engine.BuildEnv(cfg.Env, "", nil)
				cfg.Deps.Each(func(name string, dep config.Dep) {
					s := "stopped"
					if engine.CheckShell(context.Background(), ".", dep.Check, baseEnv) {
						s = "running"
					}
					statuses = append(statuses, engine.ServiceStatus{Name: name, Status: s, Type: "dep"})
				})
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
		Use:   "logs [services...]",
		Short: "Tail service logs with colored prefixes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.ValidateServiceNames(args); err != nil {
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

			engine.TailLogs(ctx, eng.LogConfigs(args), os.Stdout)
			return nil
		},
	}
}

func doctorCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check config, deps, and bootstrap status without starting anything",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			eng := newEngine(cfg)
			entries := eng.Doctor(cmd.Context())

			if jsonOutput {
				data, err := json.MarshalIndent(entries, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			for _, e := range entries {
				icon := "✓"
				if e.Status == "fail" {
					icon = "✗"
				} else if e.Status == "warn" {
					icon = "⚠"
				}
				label := e.Section
				if label == "bootstrap" && e.Status == "ok" {
					label += " (done)"
				} else if label == "bootstrap" {
					label += " (needed)"
				}
				fmt.Printf("%s %-18s %s\n", icon, e.Name, label)
				for _, w := range e.Warnings {
					fmt.Printf("  ⚠ %s\n", w)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	return cmd
}

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <service> -- <command...>",
		Short: "Run a one-off command in a service's environment",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			svc, ok := cfg.Services.Get(args[0])
			if !ok {
				return fmt.Errorf("unknown service %q", args[0])
			}
			eng := newEngine(cfg)
			env, err := eng.ServiceEnv(svc)
			if err != nil {
				return err
			}
			dir, _ := os.Getwd()
			svcDir := dir
			if svc.Dir != "" {
				svcDir = svc.Dir
			}
			return engine.RunShell(cmd.Context(), svcDir, strings.Join(args[1:], " "), env)
		},
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize deck.yaml and .gitignore",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat("deck.yaml"); os.IsNotExist(err) {
				dir, _ := os.Getwd()
				projectName := filepath.Base(dir)
				stacks := scaffold.Detect(dir)
				content := scaffold.Generate(stacks, projectName)

				if len(stacks) > 0 {
					var names []string
					for _, s := range stacks {
						label := s.Name
						if s.Dir != "." {
							label += " (" + s.Dir + ")"
						}
						names = append(names, label)
					}
					fmt.Printf("Detected: %s\n", strings.Join(names, ", "))
				}

				if err := os.WriteFile("deck.yaml", []byte(content), 0644); err != nil {
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
