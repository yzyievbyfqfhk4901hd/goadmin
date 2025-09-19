package commands

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shirou/gopsutil/v3/process"
)

type ProcessHandler struct {
	api *tgbotapi.BotAPI
}

func NewProcessHandler(api *tgbotapi.BotAPI) *ProcessHandler {
	return &ProcessHandler{
		api: api,
	}
}

type ProcessInfo struct {
	PID     int32
	Name    string
	CPU     float64
	Memory  float64
	Status  string
	User    string
	Command string
}

func (h *ProcessHandler) HandleProcessCommand(chatID int64) {
	processes, err := h.getUserProcesses()
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Error getting process information")
		h.api.Send(msg)
		return
	}

	if len(processes) == 0 {
		msg := tgbotapi.NewMessage(chatID, "No user processes found")
		h.api.Send(msg)
		return
	}

	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Memory > processes[j].Memory
	})

	maxProcesses := 15
	if len(processes) > maxProcesses {
		processes = processes[:maxProcesses]
	}

	var message strings.Builder
	message.WriteString("**Running Applications**\n\n")

	for i, proc := range processes {
		status := "+"
		if proc.Status == "stopped" || proc.Status == "zombie" {
			status = "-"
		} else if proc.Status == "sleeping" {
			status = "*"
		}

		message.WriteString(fmt.Sprintf("%d. %s **%s** (PID: %d)\n",
			i+1, status, proc.Name, proc.PID))
		message.WriteString(fmt.Sprintf("   Memory: %.1f MB | CPU: %.1f%%\n",
			proc.Memory, proc.CPU))

		if proc.Command != "" && len(proc.Command) > 50 {
			message.WriteString(fmt.Sprintf("   Command: %s...\n", proc.Command[:50]))
		} else if proc.Command != "" {
			message.WriteString(fmt.Sprintf("   Command: %s\n", proc.Command))
		}
		message.WriteString("\n")
	}

	msg := tgbotapi.NewMessage(chatID, message.String())
	msg.ParseMode = "Markdown"
	h.api.Send(msg)
}

func (h *ProcessHandler) HandleKillProcessCommand(chatID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(chatID, "Usage: /kill <PID>\nExample: /kill 1234")
		h.api.Send(msg)
		return
	}

	pidStr := parts[1]
	pid, err := strconv.ParseInt(pidStr, 10, 32)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Invalid PID. Please provide a valid number.")
		h.api.Send(msg)
		return
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Process not found or access denied.")
		h.api.Send(msg)
		return
	}

	name, _ := proc.Name()

	err = proc.Kill()
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Failed to kill process %s (PID: %d)\nError: %s", name, pid, err.Error()))
		h.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Successfully killed process %s (PID: %d)", name, pid))
	h.api.Send(msg)
}

func (h *ProcessHandler) getUserProcesses() ([]ProcessInfo, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var userProcesses []ProcessInfo
	systemProcesses := h.getSystemProcessNames()

	for _, proc := range processes {
		name, err := proc.Name()
		if err != nil {
			continue
		}

		if h.isSystemProcess(name, systemProcesses) {
			continue
		}

		cpu, _ := proc.CPUPercent()
		memInfo, _ := proc.MemoryInfo()
		status, _ := proc.Status()
		username, _ := proc.Username()
		cmdline, _ := proc.Cmdline()

		memoryMB := float64(memInfo.RSS) / 1024 / 1024

		if memoryMB > 1 || cpu > 0.1 {
			userProcesses = append(userProcesses, ProcessInfo{
				PID:     proc.Pid,
				Name:    name,
				CPU:     cpu,
				Memory:  memoryMB,
				Status:  status[0],
				User:    username,
				Command: cmdline,
			})
		}
	}

	return userProcesses, nil
}

func (h *ProcessHandler) getSystemProcessNames() []string {
	if runtime.GOOS == "windows" {
		return []string{
			"System", "smss.exe", "csrss.exe", "wininit.exe", "winlogon.exe",
			"services.exe", "lsass.exe", "svchost.exe", "dwm.exe", "explorer.exe",
			"taskhost.exe", "dllhost.exe", "conhost.exe", "audiodg.exe",
			"spoolsv.exe", "SearchIndexer.exe", "SearchProtocolHost.exe",
			"SearchFilterHost.exe", "WmiPrvSE.exe", "TrustedInstaller.exe",
			"MsMpEng.exe", "NisSrv.exe", "SecurityHealthService.exe",
			"RuntimeBroker.exe", "BackgroundTaskHost.exe", "ApplicationFrameHost.exe",
			"ShellExperienceHost.exe", "StartMenuExperienceHost.exe",
			"TextInputHost.exe", "SearchApp.exe", "LockApp.exe", "dllhost.exe",
			"WmiPrvSE.exe", "svchost.exe", "audiodg.exe", "dwm.exe",
		}
	}

	return []string{
		"init", "systemd", "kthreadd", "ksoftirqd", "migration", "rcu_",
		"rcu_sched", "rcu_bh", "rcuop_", "rcuos_", "rcuob_", "migration",
		"watchdog", "watchdogd", "ksoftirqd", "kworker", "kdevtmpfs",
		"netns", "perf", "khungtaskd", "writeback", "kintegrityd",
		"bioset", "kblockd", "ata_sff", "md", "devfreq_wq", "kworker",
		"kswapd0", "fsnotify_mark", "ecryptfs-kthrea", "kthrotld",
		"acpi_thermal_pm", "bioset", "kworker", "kdevtmpfs", "netns",
		"khungtaskd", "writeback", "kintegrityd", "bioset", "kblockd",
		"ata_sff", "md", "devfreq_wq", "kworker", "kswapd0",
	}
}

func (h *ProcessHandler) isSystemProcess(name string, systemProcesses []string) bool {
	nameLower := strings.ToLower(name)

	for _, sysProc := range systemProcesses {
		if strings.Contains(nameLower, strings.ToLower(sysProc)) {
			return true
		}
	}

	if runtime.GOOS == "windows" {
		if strings.HasPrefix(name, "conhost") ||
			strings.HasPrefix(name, "dwm") ||
			strings.HasPrefix(name, "csrss") ||
			strings.HasPrefix(name, "winlogon") ||
			strings.HasPrefix(name, "lsass") ||
			strings.HasPrefix(name, "smss") {
			return true
		}
	}

	return false
}

func (h *ProcessHandler) GetProcessList() ([]ProcessInfo, error) {
	return h.getUserProcesses()
}
