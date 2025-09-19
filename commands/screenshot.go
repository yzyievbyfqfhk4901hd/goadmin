package commands

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kbinani/screenshot"
)

type ScreenshotHandler struct {
	api *tgbotapi.BotAPI
}

func NewScreenshotHandler(api *tgbotapi.BotAPI) *ScreenshotHandler {
	return &ScreenshotHandler{
		api: api,
	}
}

func (h *ScreenshotHandler) HandleScreenshotCommand(chatID int64) {
	displays := screenshot.NumActiveDisplays()

	if displays == 0 {
		msg := tgbotapi.NewMessage(chatID, "No active displays found")
		h.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Capturing %d monitor(s)...", displays))
	h.api.Send(msg)

	screenshotDir := os.TempDir()
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to create screenshots directory")
		h.api.Send(errorMsg)
		return
	}

	successCount := 0
	var failedDisplays []int

	for i := 0; i < displays; i++ {
		bounds := screenshot.GetDisplayBounds(i)

		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			failedDisplays = append(failedDisplays, i)
			continue
		}

		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("monitor_%d_%s.png", i+1, timestamp)
		filepath := filepath.Join(screenshotDir, filename)

		file, err := os.Create(filepath)
		if err != nil {
			failedDisplays = append(failedDisplays, i)
			continue
		}

		err = png.Encode(file, img)
		file.Close()

		if err != nil {
			failedDisplays = append(failedDisplays, i)
			os.Remove(filepath)
			continue
		}

		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(filepath))
		photo.Caption = fmt.Sprintf("Monitor %d (%dx%d)", i+1, bounds.Dx(), bounds.Dy())

		_, err = h.api.Send(photo)
		if err != nil {
			failedDisplays = append(failedDisplays, i)
		} else {
			successCount++
		}

		os.Remove(filepath)
	}

	if successCount == displays {
		summaryMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Successfully captured %d monitor(s)", successCount))
		h.api.Send(summaryMsg)
	} else if successCount > 0 {
		summaryMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Captured %d of %d monitor(s). Failed: %v", successCount, displays, failedDisplays))
		h.api.Send(summaryMsg)
	} else {
		summaryMsg := tgbotapi.NewMessage(chatID, "Failed to capture any screenshots")
		h.api.Send(summaryMsg)
	}
}

func (h *ScreenshotHandler) HandleScreenshotAllCommand(chatID int64) {
	displays := screenshot.NumActiveDisplays()

	if displays == 0 {
		msg := tgbotapi.NewMessage(chatID, "No active displays found")
		h.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Capturing all monitors as single image...")
	h.api.Send(msg)

	var allBounds []image.Rectangle
	for i := 0; i < displays; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		allBounds = append(allBounds, bounds)
	}

	minX, minY := allBounds[0].Min.X, allBounds[0].Min.Y
	maxX, maxY := allBounds[0].Max.X, allBounds[0].Max.Y

	for _, bounds := range allBounds {
		if bounds.Min.X < minX {
			minX = bounds.Min.X
		}
		if bounds.Min.Y < minY {
			minY = bounds.Min.Y
		}
		if bounds.Max.X > maxX {
			maxX = bounds.Max.X
		}
		if bounds.Max.Y > maxY {
			maxY = bounds.Max.Y
		}
	}

	combinedBounds := image.Rect(minX, minY, maxX, maxY)

	img, err := screenshot.CaptureRect(combinedBounds)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to capture screenshot")
		h.api.Send(errorMsg)
		return
	}

	screenshotDir := os.TempDir()
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to create screenshots directory")
		h.api.Send(errorMsg)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("all_monitors_%s.png", timestamp)
	filepath := filepath.Join(screenshotDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to create screenshot file")
		h.api.Send(errorMsg)
		return
	}

	err = png.Encode(file, img)
	file.Close()

	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to save screenshot")
		h.api.Send(errorMsg)
		os.Remove(filepath)
		return
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(filepath))
	photo.Caption = fmt.Sprintf("All Monitors (%dx%d) - %d display(s)",
		combinedBounds.Dx(), combinedBounds.Dy(), displays)

	_, err = h.api.Send(photo)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to send screenshot")
		h.api.Send(errorMsg)
	} else {
		successMsg := tgbotapi.NewMessage(chatID, "Screenshot sent successfully")
		h.api.Send(successMsg)
	}

	os.Remove(filepath)
}

func (h *ScreenshotHandler) HandleMainMonitorCommand(chatID int64) {
	displays := screenshot.NumActiveDisplays()

	if displays == 0 {
		msg := tgbotapi.NewMessage(chatID, "No active displays found")
		h.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Capturing main monitor...")
	h.api.Send(msg)

	screenshotDir := os.TempDir()
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to create screenshots directory")
		h.api.Send(errorMsg)
		return
	}

	mainMonitorIndex := h.findMainMonitor()
	bounds := screenshot.GetDisplayBounds(mainMonitorIndex)

	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to capture main monitor")
		h.api.Send(errorMsg)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("main_monitor_%s.png", timestamp)
	filepath := filepath.Join(screenshotDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to create screenshot file")
		h.api.Send(errorMsg)
		return
	}

	err = png.Encode(file, img)
	file.Close()

	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to save screenshot")
		h.api.Send(errorMsg)
		os.Remove(filepath)
		return
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(filepath))
	photo.Caption = fmt.Sprintf("Main Monitor %d (%dx%d)", mainMonitorIndex+1, bounds.Dx(), bounds.Dy())

	_, err = h.api.Send(photo)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to send screenshot")
		h.api.Send(errorMsg)
	} else {
		successMsg := tgbotapi.NewMessage(chatID, "Main monitor screenshot sent successfully")
		h.api.Send(successMsg)
	}

	os.Remove(filepath)
}

func (h *ScreenshotHandler) GetDisplayInfo() string {
	displays := screenshot.NumActiveDisplays()

	if displays == 0 {
		return "No active displays found"
	}

	mainMonitorIndex := h.findMainMonitor()
	var info string
	info += fmt.Sprintf("**Display Information** (%d monitor(s)):\n\n", displays)

	for i := 0; i < displays; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		monitorType := ""
		if i == mainMonitorIndex {
			monitorType = " (Main)"
		}
		info += fmt.Sprintf("**Monitor %d%s:**\n", i+1, monitorType)
		info += fmt.Sprintf("• Resolution: %dx%d\n", bounds.Dx(), bounds.Dy())
		info += fmt.Sprintf("• Position: (%d, %d)\n", bounds.Min.X, bounds.Min.Y)
		info += fmt.Sprintf("• Area: %dx%d pixels\n\n", bounds.Max.X-bounds.Min.X, bounds.Max.Y-bounds.Min.Y)
	}

	return info
}

func (h *ScreenshotHandler) findMainMonitor() int {
	displays := screenshot.NumActiveDisplays()

	if displays == 0 {
		return 0
	}

	for i := 0; i < displays; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		if bounds.Min.X == 0 && bounds.Min.Y == 0 {
			return i
		}
	}

	mainIndex := 0
	minX := screenshot.GetDisplayBounds(0).Min.X

	for i := 1; i < displays; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		if bounds.Min.X < minX {
			minX = bounds.Min.X
			mainIndex = i
		}
	}

	if minX == screenshot.GetDisplayBounds(0).Min.X {
		minY := screenshot.GetDisplayBounds(0).Min.Y
		for i := 1; i < displays; i++ {
			bounds := screenshot.GetDisplayBounds(i)
			if bounds.Min.X == minX && bounds.Min.Y < minY {
				minY = bounds.Min.Y
				mainIndex = i
			}
		}
	}

	return mainIndex
}
