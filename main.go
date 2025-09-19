package main

import (
	"fmt"
	"log"
	"remoteadmin/ascii"
	"remoteadmin/bot"
	"remoteadmin/config"
	"remoteadmin/console"
)

func main() {
	ascii.DisplayDefaultArt()

	fmt.Println("> Bot starting...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	if err := cfg.ValidateConfig(); err != nil {
		log.Fatal("Invalid configuration:", err)
	}

	telegramBot, err := bot.NewBot(cfg)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	fmt.Println("> Bot started successfully!")

	consoleHandler := console.NewHandler(telegramBot, cfg)
	consoleHandler.Start()

	telegramBot.SetConsoleHandler(consoleHandler)

	fmt.Println()
	consoleHandler.ShowCommands()
	fmt.Println()

	if err := telegramBot.Start(); err != nil {
		log.Fatal("Bot error:", err)
	}
}
