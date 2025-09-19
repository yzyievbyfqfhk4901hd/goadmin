package console

import (
	"bufio"
	"fmt"
	"os"
	"remoteadmin/bot"
	"remoteadmin/config"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

type Handler struct {
	bot          *bot.Bot
	config       *config.Config
	popupChannel chan string
}

func NewHandler(bot *bot.Bot, config *config.Config) *Handler {
	return &Handler{
		bot:          bot,
		config:       config,
		popupChannel: make(chan string, 10),
	}
}

func (h *Handler) Start() {
	go h.handleConsoleInput()
	go h.handlePopups()
}

func (h *Handler) handleConsoleInput() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		h.processCommand(input)
	}
}

func (h *Handler) processCommand(input string) {
	command := strings.ToLower(input)

	switch command {
	case "1", "ping admin":
		h.pingAdmin()
	case "2", "send message":
		h.sendMessage()
	case "3", "help", "commands":
		h.showCommands()
	case "4", "exit", "quit":
		fmt.Println("> Goodbye!")
		os.Exit(0)
	default:
		fmt.Printf("> Unknown command: %s\n", input)
		fmt.Println("> Type 'help' for available commands")
	}
}

func (h *Handler) pingAdmin() {
	message := "User has called in for help"
	h.bot.SendMessageToAllAdmins(message)
	fmt.Println("> Ping sent to all admins")
}

func (h *Handler) sendMessage() {
	fmt.Print("> Enter your message: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Println("> Failed to read input")
		return
	}

	message := strings.TrimSpace(scanner.Text())
	if message == "" {
		fmt.Println("> Message cannot be empty")
		return
	}

	h.bot.SendMessageToAllAdmins(message)
	fmt.Println("> Message sent to all admins")
}

func (h *Handler) showCommands() {
	fmt.Println("Commands:")
	fmt.Println("1. Ping admin")
	fmt.Println("2. Send message")
	fmt.Println("3. help - Show this menu")
	fmt.Println("4. exit - Exit the program")
}

func (h *Handler) ShowCommands() {
	h.showCommands()
}

func (h *Handler) handlePopups() {
	for message := range h.popupChannel {
		h.showPopup(message)
	}
}

func (h *Handler) showPopup(message string) {
	fmt.Println()
	fmt.Printf("%-47s \n", message)
	fmt.Println()
	fmt.Print("> ")

	if runtime.GOOS == "windows" {
		h.showWindowsPopup(message)
	}
}

func (h *Handler) showWindowsPopup(message string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")

	title, _ := syscall.UTF16PtrFromString("Remote Admin Message")
	text, _ := syscall.UTF16PtrFromString(message)

	messageBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x1000)
}

func (h *Handler) SendPopup(message string) {
	select {
	case h.popupChannel <- message:
	default:
		fmt.Println("> Warning: Popup channel is full, message dropped")
	}
}
