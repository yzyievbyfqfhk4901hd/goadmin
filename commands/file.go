package commands

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type FileHandler struct {
	api *tgbotapi.BotAPI
}

func NewFileHandler(api *tgbotapi.BotAPI) *FileHandler {
	return &FileHandler{
		api: api,
	}
}

func (h *FileHandler) HandleFileCommand(chatID int64, message *tgbotapi.Message) {
	var file *tgbotapi.Document
	var fileName string
	var fileID string

	if message.Document != nil {
		file = message.Document
		fileName = file.FileName
		fileID = file.FileID
	} else if message.Photo != nil {
		photos := message.Photo
		photo := photos[len(photos)-1]
		file = &tgbotapi.Document{
			FileID:   photo.FileID,
			FileName: "photo.jpg",
		}
		fileName = "photo.jpg"
		fileID = photo.FileID
	} else if message.Video != nil {
		video := message.Video
		file = &tgbotapi.Document{
			FileID:   video.FileID,
			FileName: "video.mp4",
		}
		fileName = "video.mp4"
		fileID = video.FileID
	} else {
		msg := tgbotapi.NewMessage(chatID, "Please send a file, photo, or video")
		h.api.Send(msg)
		return
	}

	if !h.isAllowedFileType(fileName, file.MimeType) {
		msg := tgbotapi.NewMessage(chatID, "File type not allowed. Only images, videos, audio, and text files are permitted.")
		h.api.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Receiving file: %s", fileName))
	h.api.Send(msg)

	filePath, err := h.downloadFile(fileID, fileName)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Failed to download file: %s", err.Error()))
		h.api.Send(errorMsg)
		return
	}

	err = h.openFile(filePath)
	if err != nil {
		errorMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Failed to open file: %s", err.Error()))
		h.api.Send(errorMsg)
		os.Remove(filePath)
		return
	}

	successMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("File opened successfully: %s\n Path: %s", fileName, filePath))
	h.api.Send(successMsg)

	go func() {
		time.Sleep(30 * time.Second)
		os.Remove(filePath)
	}()
}

func (h *FileHandler) isAllowedFileType(fileName, mimeType string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))

	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff", ".ico"}
	for _, allowedExt := range imageExts {
		if ext == allowedExt {
			return true
		}
	}

	videoExts := []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".mkv", ".m4v", ".3gp"}
	for _, allowedExt := range videoExts {
		if ext == allowedExt {
			return true
		}
	}

	audioExts := []string{".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a", ".wma"}
	for _, allowedExt := range audioExts {
		if ext == allowedExt {
			return true
		}
	}

	textExts := []string{".txt", ".log", ".md", ".json", ".xml", ".csv", ".ini", ".cfg", ".conf"}
	for _, allowedExt := range textExts {
		if ext == allowedExt {
			return true
		}
	}

	if mimeType != "" {
		mediaType, _, err := mime.ParseMediaType(mimeType)
		if err == nil {
			if strings.HasPrefix(mediaType, "image/") ||
				strings.HasPrefix(mediaType, "video/") ||
				strings.HasPrefix(mediaType, "audio/") ||
				strings.HasPrefix(mediaType, "text/") {
				return true
			}
		}
	}

	return false
}

func (h *FileHandler) downloadFile(fileID, fileName string) (string, error) {
	downloadDir := os.TempDir()

	fileResp, err := h.api.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102_150405")
	localFileName := fmt.Sprintf("%s_%s", timestamp, fileName)
	localFilePath := filepath.Join(downloadDir, localFileName)

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", h.api.Token, fileResp.FilePath)

	resp, err := http.Get(fileURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	localFile, err := os.Create(localFilePath)
	if err != nil {
		return "", err
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, resp.Body)
	if err != nil {
		os.Remove(localFilePath)
		return "", err
	}

	return localFilePath, nil
}

func (h *FileHandler) openFile(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch {
	case h.isImageFile(ext):
		return h.openImageFile(filePath)
	case h.isVideoFile(ext):
		return h.openVideoFile(filePath)
	case h.isAudioFile(ext):
		return h.openAudioFile(filePath)
	case h.isTextFile(ext):
		return h.openTextFile(filePath)
	default:
		return fmt.Errorf("unsupported file type")
	}
}

func (h *FileHandler) isImageFile(ext string) bool {
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff", ".ico"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

func (h *FileHandler) isVideoFile(ext string) bool {
	videoExts := []string{".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".mkv", ".m4v", ".3gp"}
	for _, vidExt := range videoExts {
		if ext == vidExt {
			return true
		}
	}
	return false
}

func (h *FileHandler) isAudioFile(ext string) bool {
	audioExts := []string{".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a", ".wma"}
	for _, audExt := range audioExts {
		if ext == audExt {
			return true
		}
	}
	return false
}

func (h *FileHandler) isTextFile(ext string) bool {
	textExts := []string{".txt", ".log", ".md", ".json", ".xml", ".csv", ".ini", ".cfg", ".conf"}
	for _, txtExt := range textExts {
		if ext == txtExt {
			return true
		}
	}
	return false
}

func (h *FileHandler) openImageFile(filePath string) error {
	switch runtime.GOOS {
	case "windows":
		methods := []func() error{
			func() error {
				return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", filePath).Start()
			},
			func() error {
				return exec.Command("rundll32.exe", "shimgvw.dll,ImageView_Fullscreen", filePath).Start()
			},
			func() error {
				return exec.Command("mspaint.exe", filePath).Start()
			},
			func() error {
				return exec.Command("cmd", "/c", "start", "", filePath).Start()
			},
		}

		for _, method := range methods {
			if err := method(); err == nil {
				return nil
			}
		}
		return fmt.Errorf("failed to open image with any method")

	case "linux":
		viewers := []string{"xdg-open", "eog", "feh", "display", "gimp", "gwenview", "kview"}
		for _, viewer := range viewers {
			if err := exec.Command(viewer, filePath).Start(); err == nil {
				return nil
			}
		}
		return fmt.Errorf("no image viewer found")

	default:
		return fmt.Errorf("unsupported operating system")
	}
}

func (h *FileHandler) openVideoFile(filePath string) error {
	switch runtime.GOOS {
	case "windows":
		methods := []func() error{
			func() error {
				return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", filePath).Start()
			},
			func() error {
				return exec.Command("wmplayer.exe", filePath).Start()
			},
			func() error {
				return exec.Command("vlc.exe", filePath).Start()
			},
			func() error {
				return exec.Command("cmd", "/c", "start", "", filePath).Start()
			},
		}

		for _, method := range methods {
			if err := method(); err == nil {
				return nil
			}
		}
		return fmt.Errorf("failed to open video with any method")

	case "darwin":
		return exec.Command("open", "-a", "QuickTime Player", filePath).Start()

	case "linux":
		players := []string{"xdg-open", "vlc", "mplayer", "mpv", "totem", "smplayer", "kodi"}
		for _, player := range players {
			if err := exec.Command(player, filePath).Start(); err == nil {
				return nil
			}
		}
		return fmt.Errorf("no video player found")

	default:
		return fmt.Errorf("unsupported operating system")
	}
}

func (h *FileHandler) openAudioFile(filePath string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", filePath).Start()
	case "darwin":
		return exec.Command("open", "-a", "QuickTime Player", filePath).Start()
	case "linux":
		players := []string{"xdg-open", "vlc", "mplayer", "mpv", "audacious", "amarok"}
		for _, player := range players {
			if err := exec.Command(player, filePath).Start(); err == nil {
				return nil
			}
		}
		return fmt.Errorf("no audio player found")
	default:
		return fmt.Errorf("unsupported operating system")
	}
}

func (h *FileHandler) openTextFile(filePath string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("notepad.exe", filePath).Start()
	case "darwin":
		return exec.Command("open", "-a", "TextEdit", filePath).Start()
	case "linux":
		editors := []string{"xdg-open", "gedit", "kate", "mousepad", "leafpad", "nano", "vim"}
		for _, editor := range editors {
			if err := exec.Command(editor, filePath).Start(); err == nil {
				return nil
			}
		}
		return fmt.Errorf("no text editor found")
	default:
		return fmt.Errorf("unsupported operating system")
	}
}

func (h *FileHandler) GetSupportedFileTypes() string {
	return `**Supported File Types:**

**Images:**
• JPG, JPEG, PNG, GIF, BMP, WEBP, TIFF, ICO

**Videos:**
• MP4, AVI, MOV, WMV, FLV, WEBM, MKV, M4V, 3GP

**Audio:**
• MP3, WAV, OGG, FLAC, AAC, M4A, WMA

**Text:**
• TXT, LOG, MD, JSON, XML, CSV, INI, CFG, CONF

*Send any file as a document, photo, or video to auto-open it on this computer.*`
}
