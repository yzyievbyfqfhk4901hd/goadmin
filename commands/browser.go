package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"remoteadmin/config"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shirou/gopsutil/v3/process"
)

type BrowserKiller struct {
	api         *tgbotapi.BotAPI
	config      *config.Config
	bannedSites []string
	monitoring  bool
	lastKill    time.Time
}

type BannedSitesConfig struct {
	BannedSites []string `json:"banned_sites"`
}

func NewBrowserKiller(api *tgbotapi.BotAPI, cfg *config.Config) *BrowserKiller {
	bk := &BrowserKiller{
		api:        api,
		config:     cfg,
		monitoring: false,
	}
	bk.loadBannedSites()
	go bk.startAutoMonitoring()
	return bk
}

func (bk *BrowserKiller) loadBannedSites() {
	file, err := os.ReadFile("banned.json")
	if err != nil {
		fmt.Printf("Error reading banned.json: %v\n", err)
		return
	}

	var config BannedSitesConfig
	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Printf("Error parsing banned.json: %v\n", err)
		return
	}

	bk.bannedSites = config.BannedSites
}

func (bk *BrowserKiller) HandleBrowserKillerCommand(chatID int64, text string) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(chatID, "Browser Killer Commands:\n\n"+
			"/browser start - Start monitoring\n"+
			"/browser stop - Stop monitoring\n"+
			"/browser status - Check status\n"+
			"/browser list - Show banned sites")
		bk.api.Send(msg)
		return
	}

	command := parts[1]
	switch command {
	case "start":
		bk.startMonitoring(chatID)
	case "stop":
		bk.stopMonitoring(chatID)
	case "status":
		bk.showStatus(chatID)
	case "list":
		bk.showBannedSites(chatID)
	default:
		msg := tgbotapi.NewMessage(chatID, "Unknown command. Use /browser for help.")
		bk.api.Send(msg)
	}
}

func (bk *BrowserKiller) startAutoMonitoring() {
	time.Sleep(2 * time.Second)
	bk.monitoring = true
	fmt.Println("Browser Killer: Auto-started monitoring")
	go bk.monitorBrowsers()
}

func (bk *BrowserKiller) startMonitoring(chatID int64) {
	bk.monitoring = true
	msg := tgbotapi.NewMessage(chatID, "Browser monitoring started")
	bk.api.Send(msg)
	go bk.monitorBrowsers()
}

func (bk *BrowserKiller) stopMonitoring(chatID int64) {
	bk.monitoring = false
	msg := tgbotapi.NewMessage(chatID, "Browser monitoring stopped")
	bk.api.Send(msg)
}

func (bk *BrowserKiller) showStatus(chatID int64) {
	status := "Stopped"
	if bk.monitoring {
		status = "Running"
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Browser Killer Status:\nMonitoring: %s\nBanned sites: %d", status, len(bk.bannedSites)))
	bk.api.Send(msg)
}

func (bk *BrowserKiller) showBannedSites(chatID int64) {
	var message strings.Builder
	message.WriteString("Banned Sites:\n")

	for i, site := range bk.bannedSites {
		message.WriteString(fmt.Sprintf("%d. %s\n", i+1, site))
	}

	if len(bk.bannedSites) == 0 {
		message.WriteString("No sites banned")
	}

	msg := tgbotapi.NewMessage(chatID, message.String())
	bk.api.Send(msg)
}

func (bk *BrowserKiller) monitorBrowsers() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if bk.monitoring {
			bk.checkAndKillBrowsers()
		}
	}
}

func (bk *BrowserKiller) checkAndKillBrowsers() {
	processes, err := process.Processes()
	if err != nil {
		return
	}

	browserProcesses := bk.getBrowserProcesses(processes)

	for _, proc := range browserProcesses {
		if bk.isBannedSiteOpen(proc) {
			bk.killBrowserProcess(proc)
		}
	}
}

func (bk *BrowserKiller) getBrowserProcesses(processes []*process.Process) []*process.Process {
	var browsers []*process.Process

	browserNames := []string{
		"chrome.exe", "firefox.exe", "msedge.exe", "opera.exe", "brave.exe",
		"chrome", "firefox", "microsoft-edge", "opera", "brave",
		"safari", "safari.exe", "vivaldi.exe", "vivaldi",
	}

	for _, proc := range processes {
		name, err := proc.Name()
		if err != nil {
			continue
		}

		nameLower := strings.ToLower(name)
		for _, browserName := range browserNames {
			if strings.Contains(nameLower, browserName) {
				browsers = append(browsers, proc)
				break
			}
		}
	}

	return browsers
}

func (bk *BrowserKiller) isBannedSiteOpen(proc *process.Process) bool {
	title, err := proc.Name()
	if err != nil {
		return false
	}

	cmdline, err := proc.Cmdline()
	if err != nil {
		cmdline = ""
	}

	combined := strings.ToLower(title + " " + cmdline)

	for _, site := range bk.bannedSites {
		siteLower := strings.ToLower(site)
		siteLower = strings.TrimPrefix(siteLower, "https://")
		siteLower = strings.TrimPrefix(siteLower, "http://")
		siteLower = strings.TrimPrefix(siteLower, "www.")

		if strings.Contains(combined, siteLower) {
			return true
		}
	}

	return false
}

func (bk *BrowserKiller) killBrowserProcess(proc *process.Process) {
	name, _ := proc.Name()
	pid := proc.Pid

	err := proc.Kill()
	if err != nil {
		fmt.Printf("Failed to kill browser %s (PID: %d): %s\n", name, pid, err.Error())
		return
	}

	now := time.Now()
	if now.Sub(bk.lastKill) > 5*time.Second {
		fmt.Println("> Nuhuh can't view this")
		bk.lastKill = now

		go bk.notifyAdmins(fmt.Sprintf("Browser blocked: %s (PID: %d) - Banned site detected", name, pid))
	}
}

func (bk *BrowserKiller) IsSiteBanned(site string) bool {
	siteLower := strings.ToLower(site)
	for _, bannedSite := range bk.bannedSites {
		bannedLower := strings.ToLower(bannedSite)
		bannedLower = strings.TrimPrefix(bannedLower, "https://")
		bannedLower = strings.TrimPrefix(bannedLower, "http://")

		if strings.Contains(siteLower, bannedLower) {
			return true
		}
	}
	return false
}

func (bk *BrowserKiller) notifyAdmins(message string) {
	for _, userID := range bk.config.AuthorizedUsers {
		msg := tgbotapi.NewMessage(userID, message)
		bk.api.Send(msg)
	}
}
