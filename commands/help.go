package commands

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HelpHandler struct {
	api *tgbotapi.BotAPI
}

func NewHelpHandler(api *tgbotapi.BotAPI) *HelpHandler {
	return &HelpHandler{
		api: api,
	}
}

func (h *HelpHandler) HandleHelpCommand(chatID int64) {
	helpText := `**Remote Admin Bot - Commands**

**System Information:**
• /info - System hardware and software information
• /displays - Display information and resolutions

**Screenshots & Recording:**
• /ss - Screenshot each monitor separately
• /ssa - Screenshot all monitors as one image
• /ssm - Screenshot main monitor only
• /vid - Record 5-second video of main monitor (max 50MB)
• /audio - Record 10-second audio from microphone (max 50MB)

**Process Management:**
• /processes - List running applications
• /kill <PID> - Kill a process by PID

**Browser Killer:**
• /browser start - Start monitoring
• /browser stop - Stop monitoring  
• /browser status - Check status
• /browser list - Show banned sites

**Communication:**
• /msg "message" - Send message to console
• /help - Show this help menu

**File Management:**
• Send any file as document - Auto-open on this computer
• /files - Show supported file types

**Console Commands:**
• 1 - Ping admin
• 2 - Send message
• 3 - Show help
• 4 - Exit program

*All commands require authorization.*`

	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ParseMode = "Markdown"
	h.api.Send(msg)
}

func (h *HelpHandler) GetStartMessage() string {
	return "**Remote Admin Bot**\n\nWelcome! Use /help to see all available commands."
}

func (h *HelpHandler) GetUnknownCommandMessage() string {
	return "Unknown command. Use /help to see available commands."
}
