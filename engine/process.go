package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/warriorscode/deck/config"
)

type ServiceStatus struct {
	Name   string `json:"name"`
	PID    int    `json:"pid"`
	Port   int    `json:"port"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

type ProcessManager struct {
	deckDir string
	pidDir  string
	logDir  string
}

func NewProcessManager(deckDir string) *ProcessManager {
	pidDir := filepath.Join(deckDir, "pids")
	logDir := filepath.Join(deckDir, "logs")
	os.MkdirAll(pidDir, 0755)
	os.MkdirAll(logDir, 0755)
	return &ProcessManager{deckDir: deckDir, pidDir: pidDir, logDir: logDir}
}

func (pm *ProcessManager) Start(name string, svc config.Service) error {
	logFile, err := os.OpenFile(filepath.Join(pm.logDir, name+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening log file for %s: %w", name, err)
	}

	fmt.Fprintf(logFile, "--- %s | %s start ---\n", time.Now().Format("2006-01-02 15:04:05"), name)

	dir := svc.Dir
	if dir == "" {
		dir = "."
	}

	cmd := exec.Command("sh", "-c", svc.Run)
	cmd.Dir = dir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("starting %s: %w", name, err)
	}
	logFile.Close()

	pidFile := filepath.Join(pm.pidDir, name+".pid")
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		return fmt.Errorf("writing PID file for %s: %w", name, err)
	}

	go cmd.Wait()

	return nil
}

func (pm *ProcessManager) Stop(name string) error {
	pidFile := filepath.Join(pm.pidDir, name+".pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer os.Remove(pidFile)

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return nil
	}

	if f, err := os.OpenFile(filepath.Join(pm.logDir, name+".log"), os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "--- %s | %s stop ---\n", time.Now().Format("2006-01-02 15:04:05"), name)
		f.Close()
	}

	if !processAlive(pid) {
		return nil
	}

	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		syscall.Kill(pid, syscall.SIGTERM)
	}

	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			if pgid != 0 {
				syscall.Kill(-pgid, syscall.SIGKILL)
			} else {
				syscall.Kill(pid, syscall.SIGKILL)
			}
			return nil
		case <-ticker.C:
			if !processAlive(pid) {
				return nil
			}
		}
	}
}

func (pm *ProcessManager) StopAll() {
	entries, err := os.ReadDir(pm.pidDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".pid" {
			name := strings.TrimSuffix(e.Name(), ".pid")
			pm.Stop(name)
		}
	}
}

func (pm *ProcessManager) Status() []ServiceStatus {
	entries, err := os.ReadDir(pm.pidDir)
	if err != nil {
		return nil
	}
	var statuses []ServiceStatus
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".pid" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".pid")
		data, err := os.ReadFile(filepath.Join(pm.pidDir, e.Name()))
		if err != nil {
			continue
		}
		pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		status := "dead"
		if processAlive(pid) {
			status = "running"
		}
		statuses = append(statuses, ServiceStatus{Name: name, PID: pid, Status: status, Type: "service"})
	}
	return statuses
}

func (pm *ProcessManager) CheckStale() (stale, running []string) {
	entries, err := os.ReadDir(pm.pidDir)
	if err != nil {
		return nil, nil
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".pid" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".pid")
		data, _ := os.ReadFile(filepath.Join(pm.pidDir, e.Name()))
		pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		if processAlive(pid) {
			running = append(running, name)
		} else {
			stale = append(stale, name)
		}
	}
	return
}

func (pm *ProcessManager) CleanStale() {
	stale, _ := pm.CheckStale()
	for _, name := range stale {
		os.Remove(filepath.Join(pm.pidDir, name+".pid"))
	}
}

func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
