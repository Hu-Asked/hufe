package ui

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func pywalColorsPath() string {
	if dir := os.Getenv("PYWAL_CACHE_DIR"); dir != "" {
		return filepath.Join(dir, "colors.json")
	}
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "wal", "colors.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "wal", "colors.json")
}

func parsePywalScheme(data []byte) (colorScheme, error) {
	var palette struct {
		Special map[string]string `json:"special"`
		Colors  map[string]string `json:"colors"`
	}
	if err := json.Unmarshal(data, &palette); err != nil {
		return colorScheme{}, err
	}

	validated := make(map[string]lipgloss.Color, 18)
	for _, name := range []string{"background", "foreground"} {
		color, err := hexColor(palette.Special[name])
		if err != nil {
			return colorScheme{}, fmt.Errorf("pywal special.%s: %w", name, err)
		}
		validated[name] = color
	}
	for index := range 16 {
		name := fmt.Sprintf("color%d", index)
		color, err := hexColor(palette.Colors[name])
		if err != nil {
			return colorScheme{}, fmt.Errorf("pywal colors.%s: %w", name, err)
		}
		validated[name] = color
	}

	foreground := validated["foreground"]
	background := validated["background"]
	accent := validated["color4"]
	return colorScheme{
		HeaderForeground:          readableOn(accent, foreground, background),
		HeaderBackground:          accent,
		BoxBorder:                 validated["color8"],
		ListItemForeground:        foreground,
		ListItemDimForeground:     validated["color8"],
		ListCursorIndicator:       accent,
		ListSelectedForeground:    readableOn(accent, foreground, background),
		ListSelectedBackground:    accent,
		ListFilterMatchForeground: validated["color6"],
		PathForeground:            accent,
		KeyForeground:             validated["color5"],
		HintForeground:            validated["color8"],
		StatusForeground:          validated["color8"],
		StatusErrorForeground:     validated["color1"],
	}, nil
}

func hexColor(value string) (lipgloss.Color, error) {
	if len(value) != 7 || value[0] != '#' {
		return "", fmt.Errorf("expected #RRGGBB, got %q", value)
	}
	if _, err := strconv.ParseUint(value[1:], 16, 24); err != nil {
		return "", fmt.Errorf("invalid hex color %q", value)
	}
	return lipgloss.Color(strings.ToLower(value)), nil
}

func readableOn(background, first, second lipgloss.Color) lipgloss.Color {
	if contrastRatio(background, first) >= contrastRatio(background, second) {
		return first
	}
	return second
}

func contrastRatio(a, b lipgloss.Color) float64 {
	first, second := luminance(a), luminance(b)
	if first < second {
		first, second = second, first
	}
	return (first + 0.05) / (second + 0.05)
}

func luminance(color lipgloss.Color) float64 {
	hex := string(color)[1:]
	channels := [3]float64{}
	for index := range channels {
		value, _ := strconv.ParseUint(hex[index*2:index*2+2], 16, 8)
		channel := float64(value) / 255
		if channel <= 0.04045 {
			channels[index] = channel / 12.92
		} else {
			channels[index] = math.Pow((channel+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
}

func (m *Model) refreshPywalScheme() bool {
	if m.pywalPath == "" {
		return false
	}
	info, err := os.Stat(m.pywalPath)
	if err != nil {
		m.pywalSeen = false
		return false
	}
	if m.pywalSeen && info.Size() == m.pywalInfo.Size() && info.ModTime().Equal(m.pywalInfo.ModTime()) && os.SameFile(info, m.pywalInfo) {
		return false
	}
	data, err := os.ReadFile(m.pywalPath)
	if err != nil {
		return false
	}
	m.pywalSeen = true
	m.pywalInfo = info
	scheme, err := parsePywalScheme(data)
	if err != nil {
		return false
	}
	m.applyScheme(scheme)
	return true
}

func (m *Model) applyScheme(scheme colorScheme) {
	applyColorScheme(scheme)
	m.list.Styles.Title = headerTitleStyle
	m.list.Styles.TitleBar = headerBarStyle.Width(m.boxWidth)
	m.list.SetDelegate(relativeLineDelegate{styles: itemStyles()})
	styleTextInput(&m.searchInput)
	if m.rename != nil {
		styleTextInput(&m.rename.input)
	}
	if m.creation != nil {
		styleTextInput(&m.creation.input)
	}
}

type pywalTickMsg time.Time

func nextPywalTick() tea.Cmd {
	return tea.Tick(time.Second, func(now time.Time) tea.Msg {
		return pywalTickMsg(now)
	})
}
