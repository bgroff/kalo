package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ThemeIcons defines all icons/symbols used in the application
type ThemeIcons struct {
	FolderClosed    string `json:"folderClosed"`    // Closed folder icon
	FolderOpen      string `json:"folderOpen"`      // Open folder icon
	File            string `json:"file"`            // File icon
	Request         string `json:"request"`         // Request file icon
	Collection      string `json:"collection"`      // Collection icon
	ActiveIndicator string `json:"activeIndicator"` // Active/selected indicator (▶)
	Cursor          string `json:"cursor"`          // Text cursor character
	ListBullet      string `json:"listBullet"`      // List item bullet
}

// ThemeColors defines all colors used in the application
type ThemeColors struct {
	Primary     string `json:"primary"`     // Main accent color
	Secondary   string `json:"secondary"`   // Secondary accent color
	Background  string `json:"background"`  // Main background
	Surface     string `json:"surface"`     // Panel backgrounds
	Border      string `json:"border"`      // Border colors
	BorderFocus string `json:"borderFocus"` // Focused border colors
	Text        string `json:"text"`        // Main text color
	TextMuted   string `json:"textMuted"`   // Muted text color
	TextAccent  string `json:"textAccent"`  // Accent text color
	Success     string `json:"success"`     // Success/OK colors
	Warning     string `json:"warning"`     // Warning colors
	Error       string `json:"error"`       // Error colors
	Cursor      string `json:"cursor"`      // Cursor/selection color
}

// ThemeConfig represents the complete theme configuration
type ThemeConfig struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Colors      ThemeColors `json:"colors"`
	Icons       ThemeIcons  `json:"icons"`
}

// Theme holds the active theme and provides styled components
type Theme struct {
	Config ThemeConfig
	
	// Pre-built styles for common UI elements
	FocusedStyle     lipgloss.Style
	BlurredStyle     lipgloss.Style
	TitleStyle       lipgloss.Style
	CursorStyle      lipgloss.Style
	TextCursorStyle  lipgloss.Style
	MethodStyle      lipgloss.Style
	URLStyle         lipgloss.Style
	SectionStyle     lipgloss.Style
	HeaderStyle      lipgloss.Style
	StatusOkStyle    lipgloss.Style
	ErrorStyle       lipgloss.Style
	WarningStyle     lipgloss.Style
	BackgroundStyle  lipgloss.Style
}

// defaultTheme provides fallback values
func defaultTheme() ThemeConfig {
	return ThemeConfig{
		Name:        "Default",
		Description: "Default Kalo theme",
		Colors: ThemeColors{
			Primary:     "62",   // Purple/blue
			Secondary:   "86",   // Green
			Background:  "235",  // Dark gray
			Surface:     "236",  // Slightly lighter gray
			Border:      "240",  // Gray border
			BorderFocus: "62",   // Purple border when focused
			Text:        "230",  // Light text
			TextMuted:   "241",  // Muted text
			TextAccent:  "86",   // Green accent text
			Success:     "46",   // Green
			Warning:     "220",  // Yellow
			Error:       "196",  // Red
			Cursor:      "212",  // Pink/purple cursor
		},
		Icons: ThemeIcons{
			FolderClosed:    "📁",  // Folder icon
			FolderOpen:      "📂",  // Open folder icon
			File:            "📄",  // File icon
			Request:         "📝",  // Request file icon
			Collection:      "📚",  // Collection icon
			ActiveIndicator: "▶",   // Active indicator
			Cursor:          "█",   // Block cursor
			ListBullet:      "•",   // List bullet
		},
	}
}

// NewTheme creates a new theme instance with styles
func NewTheme(config ThemeConfig) *Theme {
	theme := &Theme{Config: config}
	theme.buildStyles()
	return theme
}

// buildStyles creates all the lipgloss styles from the theme colors
func (t *Theme) buildStyles() {
	colors := t.Config.Colors
	
	t.FocusedStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colors.BorderFocus))
	
	t.BlurredStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colors.Border))
	
	t.TitleStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(colors.Primary)).
		Foreground(lipgloss.Color(colors.Text)).
		Padding(0, 1)
	
	t.CursorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Cursor)).
		Bold(true)
	
	t.TextCursorStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(colors.Cursor)).
		Foreground(lipgloss.Color("0"))
	
	t.MethodStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(colors.Success)).
		Foreground(lipgloss.Color("0")).
		Padding(0, 1).
		Bold(true)
	
	t.URLStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Secondary))
	
	t.SectionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.TextAccent))
	
	t.HeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.TextMuted)).
		MarginTop(1)
	
	t.StatusOkStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(colors.Success)).
		Foreground(lipgloss.Color("0")).
		Padding(0, 1)
	
	t.ErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Error))
	
	t.WarningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Warning))
	
	t.BackgroundStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(colors.Background))
}

// LoadTheme loads a theme from file with fallback to default
func LoadTheme(themeName string) *Theme {
	// Try to load from user config first
	if config, err := loadThemeFromFile(themeName); err == nil {
		return NewTheme(config)
	}
	
	// Fallback to default theme
	return NewTheme(defaultTheme())
}

// loadThemeFromFile attempts to load theme from various locations
func loadThemeFromFile(themeName string) (ThemeConfig, error) {
	// Try user config directory first
	configDir, err := getConfigDir()
	if err == nil {
		themePath := filepath.Join(configDir, "themes", themeName+".json")
		if config, err := readThemeFile(themePath); err == nil {
			return config, nil
		}
	}
	
	// Try built-in themes
	builtinPath := filepath.Join("themes", themeName+".json")
	if config, err := readThemeFile(builtinPath); err == nil {
		return config, nil
	}
	
	return ThemeConfig{}, os.ErrNotExist
}

// readThemeFile reads and parses a theme file
func readThemeFile(path string) (ThemeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ThemeConfig{}, err
	}
	
	var config ThemeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ThemeConfig{}, err
	}
	
	// Merge with default theme to fill any missing values
	return mergeWithDefault(config), nil
}

// mergeWithDefault fills missing theme values with defaults
func mergeWithDefault(theme ThemeConfig) ThemeConfig {
	defaultConfig := defaultTheme()
	
	// If any color is empty, use default
	if theme.Colors.Primary == "" {
		theme.Colors.Primary = defaultConfig.Colors.Primary
	}
	if theme.Colors.Secondary == "" {
		theme.Colors.Secondary = defaultConfig.Colors.Secondary
	}
	if theme.Colors.Background == "" {
		theme.Colors.Background = defaultConfig.Colors.Background
	}
	if theme.Colors.Surface == "" {
		theme.Colors.Surface = defaultConfig.Colors.Surface
	}
	if theme.Colors.Border == "" {
		theme.Colors.Border = defaultConfig.Colors.Border
	}
	if theme.Colors.BorderFocus == "" {
		theme.Colors.BorderFocus = defaultConfig.Colors.BorderFocus
	}
	if theme.Colors.Text == "" {
		theme.Colors.Text = defaultConfig.Colors.Text
	}
	if theme.Colors.TextMuted == "" {
		theme.Colors.TextMuted = defaultConfig.Colors.TextMuted
	}
	if theme.Colors.TextAccent == "" {
		theme.Colors.TextAccent = defaultConfig.Colors.TextAccent
	}
	if theme.Colors.Success == "" {
		theme.Colors.Success = defaultConfig.Colors.Success
	}
	if theme.Colors.Warning == "" {
		theme.Colors.Warning = defaultConfig.Colors.Warning
	}
	if theme.Colors.Error == "" {
		theme.Colors.Error = defaultConfig.Colors.Error
	}
	if theme.Colors.Cursor == "" {
		theme.Colors.Cursor = defaultConfig.Colors.Cursor
	}
	
	// If any icon is empty, use default
	if theme.Icons.FolderClosed == "" {
		theme.Icons.FolderClosed = defaultConfig.Icons.FolderClosed
	}
	if theme.Icons.FolderOpen == "" {
		theme.Icons.FolderOpen = defaultConfig.Icons.FolderOpen
	}
	if theme.Icons.File == "" {
		theme.Icons.File = defaultConfig.Icons.File
	}
	if theme.Icons.Request == "" {
		theme.Icons.Request = defaultConfig.Icons.Request
	}
	if theme.Icons.Collection == "" {
		theme.Icons.Collection = defaultConfig.Icons.Collection
	}
	if theme.Icons.ActiveIndicator == "" {
		theme.Icons.ActiveIndicator = defaultConfig.Icons.ActiveIndicator
	}
	if theme.Icons.Cursor == "" {
		theme.Icons.Cursor = defaultConfig.Icons.Cursor
	}
	if theme.Icons.ListBullet == "" {
		theme.Icons.ListBullet = defaultConfig.Icons.ListBullet
	}
	
	if theme.Name == "" {
		theme.Name = defaultConfig.Name
	}
	if theme.Description == "" {
		theme.Description = defaultConfig.Description
	}
	
	return theme
}

// getConfigDir returns the user configuration directory
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	configDir := filepath.Join(homeDir, ".kalo")
	
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	
	return configDir, nil
}

// SaveTheme saves a theme configuration to file
func (t *Theme) SaveTheme(name string) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}
	
	themesDir := filepath.Join(configDir, "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return err
	}
	
	themePath := filepath.Join(themesDir, name+".json")
	
	data, err := json.MarshalIndent(t.Config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(themePath, data, 0644)
}

// GetAvailableThemes returns list of available theme names
func GetAvailableThemes() []string {
	themes := []string{"default"} // Always include default
	
	// Check built-in themes first
	if entries, err := os.ReadDir("themes"); err == nil {
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".json" {
				name := strings.TrimSuffix(entry.Name(), ".json")
				if name != "default" { // Don't duplicate default
					themes = append(themes, name)
				}
			}
		}
	}
	
	// Check user themes
	configDir, err := getConfigDir()
	if err == nil {
		themesDir := filepath.Join(configDir, "themes")
		if entries, err := os.ReadDir(themesDir); err == nil {
			for _, entry := range entries {
				if filepath.Ext(entry.Name()) == ".json" {
					name := strings.TrimSuffix(entry.Name(), ".json")
					// Avoid duplicates from built-in themes
					exists := false
					for _, existing := range themes {
						if existing == name {
							exists = true
							break
						}
					}
					if !exists {
						themes = append(themes, name)
					}
				}
			}
		}
	}
	
	return themes
}

// ReloadTheme reloads the current theme or switches to a new one
func ReloadTheme(themeName string) *Theme {
	return LoadTheme(themeName)
}

// GetCurrentThemeName returns the name of the currently active theme
func GetCurrentThemeName() string {
	if currentTheme == nil {
		return "default"
	}
	return currentTheme.Config.Name
}