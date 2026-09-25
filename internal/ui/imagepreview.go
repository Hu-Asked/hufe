package ui

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const (
	maxImageBytes  = 50 << 20
	maxImagePixels = 40_000_000
	maxImageSide   = 1024
)

type imagePreviewMsg struct {
	path string
	seq  uint64
	png  []byte
	err  error
}

func isImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tif", ".tiff":
		return true
	}
	return false
}

func newKittyImageID() uint32 {
	var data [4]byte
	if _, err := rand.Read(data[:]); err == nil {
		if id := binary.BigEndian.Uint32(data[:]); id != 0 {
			return id
		}
	}
	return uint32(os.Getpid()) + 1
}

func (m *Model) startImagePreview(path string) {
	m.previewImage = true
	if !m.kittyGraphics {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.previewImageCancel = cancel
	ch := m.previewImageCh
	seq := m.previewImageSeq
	go func() {
		png, err := decodePreviewImage(ctx, path)
		select {
		case ch <- imagePreviewMsg{path: path, seq: seq, png: png, err: err}:
		case <-ctx.Done():
		}
	}()
}

func (m *Model) stopImagePreview() {
	m.previewImageSeq++
	if m.previewImageCancel != nil {
		m.previewImageCancel()
		m.previewImageCancel = nil
	}
	if m.previewImageFile != "" {
		_ = os.Remove(m.previewImageFile)
		m.previewImageFile = ""
	}
	m.previewImage = false
}

func (m *Model) acceptImagePreview(msg imagePreviewMsg) {
	if msg.path != m.previewPath || msg.seq != m.previewImageSeq || !m.previewImage {
		return
	}
	if m.previewImageCancel != nil {
		m.previewImageCancel()
		m.previewImageCancel = nil
	}
	if msg.err != nil {
		m.previewErr = msg.err
		return
	}
	f, err := os.CreateTemp("", "hufe-preview-*.png")
	if err != nil {
		m.previewErr = err
		return
	}
	if _, err = f.Write(msg.png); err == nil {
		err = f.Close()
	} else {
		_ = f.Close()
	}
	if err != nil {
		_ = os.Remove(f.Name())
		m.previewErr = err
		return
	}
	m.previewImageFile = f.Name()
}

func waitForImagePreview(ch <-chan imagePreviewMsg) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func decodePreviewImage(ctx context.Context, path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxImageBytes {
		return nil, fmt.Errorf("image is not a regular file or exceeds %d MiB", maxImageBytes>>20)
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		return decodeNativePreviewImage(ctx, path)
	default:
		return decodeMagickPreviewImage(ctx, path)
	}
}

func decodeNativePreviewImage(ctx context.Context, path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	config, _, err := image.DecodeConfig(f)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxImagePixels {
		return nil, fmt.Errorf("image dimensions exceed preview limit")
	}
	f, err = os.Open(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return encodePreviewPNG(ctx, img)
}

func encodePreviewPNG(ctx context.Context, img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w > maxImageSide || h > maxImageSide {
		scale := float64(maxImageSide) / float64(max(w, h))
		nw, nh := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
		resized := image.NewRGBA(image.Rect(0, 0, nw, nh))
		for y := 0; y < nh; y++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			for x := 0; x < nw; x++ {
				resized.Set(x, y, img.At(bounds.Min.X+x*w/nw, bounds.Min.Y+y*h/nh))
			}
		}
		img = resized
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeMagickPreviewImage(ctx context.Context, path string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "magick", path+"[0]", "-auto-orient", "-resize", "1024x1024>", "png:-")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.Error); ok {
			return nil, fmt.Errorf("ImageMagick is required for this image format")
		}
		return nil, fmt.Errorf("unable to decode image: %s", strings.TrimSpace(stderr.String()))
	}
	return data, nil
}

func (m *Model) kittyImageSuffix(show bool) string {
	if !m.kittyGraphics {
		return ""
	}
	id := fmt.Sprint(m.kittyImageID)
	seq := ansi.KittyGraphics(nil, "a=d", "d=I", "i="+id, "q=2")
	if show && m.previewImageFile != "" && m.previewWidth > 0 && m.previewHeight > 2 {
		cols, rows := m.previewWidth, m.previewHeight-2
		left, top := m.boxWidth+5, 4 // 1-based coordinates inside the preview border/header.
		path := base64.StdEncoding.EncodeToString([]byte(m.previewImageFile))
		seq += fmt.Sprintf("\x1b[%d;%dH", top, left)
		seq += ansi.KittyGraphics([]byte(path), "a=T", "f=100", "t=f", "i="+id,
			fmt.Sprintf("c=%d", cols), fmt.Sprintf("r=%d", rows), "C=1", "q=2")
	}
	return "\x1b7" + seq + "\x1b8"
}

// Close releases temporary image data after Bubble Tea exits.
func (m *Model) Close() {
	m.stopImagePreview()
}

// KittyImageClear removes Hufe's image after Bubble Tea leaves the alternate screen.
func (m *Model) KittyImageClear() string {
	return m.kittyImageSuffix(false)
}
