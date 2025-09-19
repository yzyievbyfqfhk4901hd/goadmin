package commands

import (
	"remoteadmin/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageHandler struct {
	api    *tgbotapi.BotAPI
	config *config.Config
}

func NewMessageHandler(api *tgbotapi.BotAPI, cfg *config.Config) *MessageHandler {
	return &MessageHandler{
		api:    api,
		config: cfg,
	}
}

func (h *MessageHandler) SendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	h.api.Send(msg)
}

func (h *MessageHandler) SendMessageToAllAdmins(text string) {
	for _, userID := range h.config.AuthorizedUsers {
		h.SendMessage(userID, text)
	}
}
