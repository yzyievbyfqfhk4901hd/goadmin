package commands

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MsgHandler struct {
	api            *tgbotapi.BotAPI
	consoleHandler interface {
		SendPopup(message string)
	}
}

func NewMsgHandler(api *tgbotapi.BotAPI, consoleHandler interface {
	SendPopup(message string)
}) *MsgHandler {
	return &MsgHandler{
		api:            api,
		consoleHandler: consoleHandler,
	}
}

func (h *MsgHandler) HandleMsgCommand(chatID int64, text string, userName string) {
	message := strings.TrimSpace(strings.TrimPrefix(text, "/msg"))

	if message == "" {
		msg := tgbotapi.NewMessage(chatID, "Usage: /msg \"your message here\"")
		h.api.Send(msg)
		return
	}

	if h.consoleHandler != nil {
		formattedMessage := userName + ": " + message
		h.consoleHandler.SendPopup(formattedMessage)
	}

	confirmMsg := tgbotapi.NewMessage(chatID, "Message sent to console!")
	h.api.Send(confirmMsg)
}
