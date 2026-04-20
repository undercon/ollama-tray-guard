package guard

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// noWindow returns an exec.Cmd with CREATE_NO_WINDOW set so no console pops up.
func noWindow(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return cmd
}

type GPUInfo struct {
	UsedMB  float64
	TotalMB float64
}

type ProcessInfo struct {
	PID    int
	UsedMB float64
}

func QueryGPU() (GPUInfo, error) {
	out, err := noWindow("nvidia-smi",
		"--query-gpu=memory.used,memory.total",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return GPUInfo{}, fmt.Errorf("nvidia-smi: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) < 2 {
		return GPUInfo{}, fmt.Errorf("unexpected nvidia-smi output: %s", out)
	}
	used, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	total, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	return GPUInfo{UsedMB: used, TotalMB: total}, nil
}

func QueryProcesses() ([]ProcessInfo, error) {
	out, err := noWindow("nvidia-smi",
		"--query-compute-apps=pid,used_memory",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi: %w", err)
	}
	var procs []ProcessInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "no running") {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		pid, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		mem, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		procs = append(procs, ProcessInfo{PID: pid, UsedMB: mem})
	}
	return procs, nil
}

// GetOllamaPIDs returns PIDs of ollama processes
func GetOllamaPIDs() map[int]bool {
	pids := make(map[int]bool)
	out, err := noWindow("tasklist", "/FI", "IMAGENAME eq ollama*", "/FO", "CSV", "/NH").Output()
	if err != nil {
		// Fallback: wmic
		out2, err2 := noWindow("wmic", "process", "where", "name like '%ollama%'", "get", "processid").Output()
		if err2 != nil {
			return pids
		}
		for _, line := range strings.Split(string(out2), "\n") {
			line = strings.TrimSpace(line)
			if pid, err := strconv.Atoi(line); err == nil {
				pids[pid] = true
			}
		}
		return pids
	}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			pidStr := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			if pid, err := strconv.Atoi(pidStr); err == nil {
				pids[pid] = true
			}
		}
	}
	return pids
}

// NonOllamaVRAM returns VRAM in MB used by non-Ollama processes
func NonOllamaVRAM() (float64, error) {
	gpu, err := QueryGPU()
	if err != nil {
		return 0, err
	}
	procs, err := QueryProcesses()
	if err != nil {
		return gpu.UsedMB, nil
	}
	ollamaPIDs := GetOllamaPIDs()
	var ollamaVRAM float64
	for _, p := range procs {
		if ollamaPIDs[p.PID] {
			ollamaVRAM += p.UsedMB
		}
	}
	return gpu.UsedMB - ollamaVRAM, nil
}
