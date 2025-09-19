package commands

import (
	"fmt"
	"os"
	"remoteadmin/config"
	"remoteadmin/hardware"
	"runtime"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type InfoHandler struct {
	api       *tgbotapi.BotAPI
	config    *config.Config
	startTime time.Time
}

func NewInfoHandler(api *tgbotapi.BotAPI, cfg *config.Config, startTime time.Time) *InfoHandler {
	return &InfoHandler{
		api:       api,
		config:    cfg,
		startTime: startTime,
	}
}

func (h *InfoHandler) HandleInfoCommand(chatID int64) {
	uptime := time.Since(h.startTime)

	hardwareInfo := hardware.GetHardwareInfo()

	hostname, _ := os.Hostname()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	msg1 := tgbotapi.NewMessage(chatID, fmt.Sprintf("Remote Admin Bot Info\n\n%s", hardwareInfo))
	_, err := h.api.Send(msg1)
	if err != nil {
		simpleMsg := tgbotapi.NewMessage(chatID, "Remote Admin Bot Info\n\n Error loading hardware information")
		h.api.Send(simpleMsg)
	}

	botInfo := fmt.Sprintf(`*Bot Information:*
• Hostname: %s
• Uptime: %s
• Authorized Users: %d
• Bot Username: @%s
• Process Memory: %d MB
• Goroutines: %d

*Last Updated:* %s`,
		hostname,
		formatUptime(uptime),
		len(h.config.AuthorizedUsers),
		h.api.Self.UserName,
		memStats.Alloc/1024/1024,
		runtime.NumGoroutine(),
		time.Now().Format("2006-01-02 15:04:05 MST"))

	msg2 := tgbotapi.NewMessage(chatID, botInfo)
	msg2.ParseMode = "Markdown"
	h.api.Send(msg2)
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
