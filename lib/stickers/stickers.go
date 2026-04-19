package stickers

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type StickerMeta struct {
	Packname string
	Author   string
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func toWebp(media []byte, mediaType string) ([]byte, error) {
	ext := "jpg"
	if mediaType == "video" {
		ext = "mp4"
	}
	tmpIn := filepath.Join(os.TempDir(), randomHex(6)+"."+ext)
	tmpOut := filepath.Join(os.TempDir(), randomHex(6)+".webp")
	defer os.Remove(tmpIn)
	defer os.Remove(tmpOut)

	if err := os.WriteFile(tmpIn, media, 0644); err != nil {
		return nil, err
	}

	var args []string
	if mediaType == "video" {
		args = []string{
			"-i", tmpIn,
			"-vcodec", "libwebp",
			"-vf", "scale=320:320:force_original_aspect_ratio=increase,crop=320:320,fps=15",
			"-loop", "0",
			"-ss", "00:00:00",
			"-t", "00:00:05",
			"-preset", "default",
			"-an", "-vsync", "0",
			tmpOut,
		}
	} else {
		args = []string{
			"-i", tmpIn,
			"-vcodec", "libwebp",
			"-vf", "scale=320:320:force_original_aspect_ratio=increase,crop=320:320",
			tmpOut,
		}
	}

	cmd := exec.Command("ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg error: %v — %s", err, stderr.String())
	}

	return os.ReadFile(tmpOut)
}

func ImageToWebp(media []byte) ([]byte, error) {
	return toWebp(media, "image")
}

func VideoToWebp(media []byte) ([]byte, error) {
	return toWebp(media, "video")
}

func writeExifWebp(webpBuf []byte, meta StickerMeta) ([]byte, error) {
	type exifJSON struct {
		PackID        string   `json:"sticker-pack-id"`
		PackName      string   `json:"sticker-pack-name"`
		PackPublisher string   `json:"sticker-pack-publisher"`
		Emojis        []string `json:"emojis"`
	}

	jsonData, _ := json.Marshal(exifJSON{
		PackID:        "Lurus-Bot-" + randomHex(4),
		PackName:      meta.Packname,
		PackPublisher: meta.Author,
		Emojis:        []string{"🍧"},
	})

	exifAttr := []byte{
		0x49, 0x49, 0x2a, 0x00, 0x08, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x41, 0x57, 0x07, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x16, 0x00, 0x00, 0x00,
	}
	binary.LittleEndian.PutUint32(exifAttr[14:], uint32(len(jsonData)))
	exif := append(exifAttr, jsonData...)

	tmpIn := filepath.Join(os.TempDir(), randomHex(6)+".webp")
	tmpOut := filepath.Join(os.TempDir(), randomHex(6)+".webp")
	defer os.Remove(tmpIn)
	defer os.Remove(tmpOut)

	if err := os.WriteFile(tmpIn, webpBuf, 0644); err != nil {
		return nil, err
	}

	cmd := exec.Command("python3", "-c", buildExifScript(tmpIn, tmpOut, exif))
	if err := cmd.Run(); err != nil {
		cmd2 := exec.Command("webpmux", "-set", "exif", "-", tmpIn, "-o", tmpOut)
		cmd2.Stdin = bytes.NewReader(exif)
		if err2 := cmd2.Run(); err2 != nil {
			return webpBuf, nil
		}
	}

	result, err := os.ReadFile(tmpOut)
	if err != nil {
		return webpBuf, nil
	}
	return result, nil
}

func buildExifScript(inPath, outPath string, exif []byte) string {
	exifHex := hex.EncodeToString(exif)
	return fmt.Sprintf(`
import struct, sys
with open('%s','rb') as f: data=f.read()
exif=bytes.fromhex('%s')
riff=b'RIFF'+struct.pack('<I',len(data)+len(exif)+10)+b'WEBP'
exif_chunk=b'EXIF'+struct.pack('<I',len(exif))+exif
with open('%s','wb') as f: f.write(riff+data[12:]+exif_chunk)
`, inPath, exifHex, outPath)
}

func WriteExifImg(media []byte, meta StickerMeta) ([]byte, error) {
	webp, err := ImageToWebp(media)
	if err != nil {
		return nil, err
	}
	return writeExifWebp(webp, meta)
}

func WriteExifVid(media []byte, meta StickerMeta) ([]byte, error) {
	webp, err := VideoToWebp(media)
	if err != nil {
		return nil, err
	}
	return writeExifWebp(webp, meta)
}
