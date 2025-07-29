package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InputType int

const (
	TextInput InputType = iota
	FileInput
	ConfirmInput
	MethodURLInput
	OpenAPIImportInput
	ThemeSelectionInput
)

type InputSpec struct {
	Type        InputType
	Title       string
	Prompt      string
	Placeholder string
	Action      string
	ActionData  map[string]interface{}
	IsEdit      bool                   // Whether this is editing an existing item
	PreFill     map[string]interface{} // Pre-filled values for editing
}

type InputDialog struct {
	visible         bool
	spec            InputSpec
	textInput       textinput.Model
	urlInput        textinput.Model
	nameInput       textinput.Model
	tagsInput       textinput.Model
	collectionInput textinput.Model
	filePicker      filepicker.Model
	confirmed       bool
	selectedMethod  int
	methods         []string
	currentField    int // 0 = method, 1 = name, 2 = url, 3 = tags (for MethodURLInput) OR 0 = url/file, 1 = collection (for OpenAPIImportInput)
	selectedFile    string
	useFilePicker   bool // whether to show file picker or URL input for OpenAPI
	// Theme selection fields
	themes          []string
	selectedTheme   int
}

func NewInputDialog() *InputDialog {
	// Create text input for general use
	ti := textinput.New()
	ti.Focus()
	
	// Create name input for request display name
	nameTi := textinput.New()
	nameTi.Placeholder = "Get Users"
	nameTi.Width = 50
	
	// Create URL input for method/URL dialog
	urlTi := textinput.New()
	urlTi.Placeholder = "https://api.example.com/endpoint"
	urlTi.Width = 50
	
	// Create tags input for method/URL dialog
	tagsTi := textinput.New()
	tagsTi.Placeholder = "tag1, tag2, tag3"
	tagsTi.Width = 50
	
	// Create collection input for OpenAPI import
	collectionTi := textinput.New()
	collectionTi.Placeholder = "my-api-collection"
	collectionTi.Width = 50
	
	// Create file picker for OpenAPI import
	fp := filepicker.New()
	fp.AllowedTypes = []string{".json", ".yaml", ".yml"}
	fp.CurrentDirectory, _ = os.UserHomeDir()
	fp.DirAllowed = true  // Allow directory navigation
	fp.FileAllowed = true // Allow file selection
	fp.ShowHidden = false // Don't show hidden files by default
	fp.SetHeight(15)      // Set reasonable height for dialog
	
	return &InputDialog{
		visible:         false,
		textInput:       ti,
		nameInput:       nameTi,
		urlInput:        urlTi,
		tagsInput:       tagsTi,
		collectionInput: collectionTi,
		filePicker:      fp,
		methods:         []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "QUERY"},
		selectedMethod:  0,
		currentField:    0,
		useFilePicker:   true, // Default to file picker for OpenAPI
		themes:          GetAvailableThemes(),
		selectedTheme:   0,
	}
}

func (id *InputDialog) Show(spec InputSpec) {
	id.visible = true
	id.spec = spec
	id.confirmed = false
	id.selectedMethod = 0
	id.currentField = 0
	
	// Reset and configure text inputs based on type
	id.textInput.SetValue("")
	id.nameInput.SetValue("")
	id.urlInput.SetValue("")
	id.tagsInput.SetValue("")
	id.collectionInput.SetValue("")
	
	// Pre-fill values if this is an edit operation
	if spec.IsEdit && spec.PreFill != nil {
		if spec.Type == MethodURLInput {
			// Pre-fill method, name, and URL for request editing
			if method, ok := spec.PreFill["method"].(string); ok {
				// Find the method index
				for i, m := range id.methods {
					if m == method {
						id.selectedMethod = i
						break
					}
				}
			}
			if name, ok := spec.PreFill["name"].(string); ok {
				id.nameInput.SetValue(name)
			}
			if url, ok := spec.PreFill["url"].(string); ok {
				id.urlInput.SetValue(url)
			}
			if tags, ok := spec.PreFill["tags"].([]string); ok {
				id.tagsInput.SetValue(strings.Join(tags, ", "))
			}
		} else {
			// Pre-fill general text input
			if value, ok := spec.PreFill["value"].(string); ok {
				id.textInput.SetValue(value)
			}
		}
	}
	
	if spec.Type == MethodURLInput {
		id.textInput.Blur()
		id.nameInput.Focus() // Start with name field focused
		id.urlInput.Blur()
		id.tagsInput.Blur()
		id.collectionInput.Blur()
	} else if spec.Type == OpenAPIImportInput {
		id.textInput.Blur()
		id.nameInput.Blur()
		id.urlInput.Blur()
		id.tagsInput.Blur()
		id.collectionInput.Blur()
		id.useFilePicker = true // Default to file picker
		id.selectedFile = ""
		id.textInput.Placeholder = "URL to OpenAPI spec"
		// Re-initialize file picker to current directory
		homeDir, _ := os.UserHomeDir()
		id.filePicker.CurrentDirectory = homeDir
	} else if spec.Type == ThemeSelectionInput {
		// Theme selection - no text input needed
		id.textInput.Blur()
		id.nameInput.Blur()
		id.urlInput.Blur()
		id.tagsInput.Blur()
		id.collectionInput.Blur()
		// Find current theme in list
		currentThemeName := GetCurrentThemeName()
		for i, theme := range id.themes {
			if theme == currentThemeName {
				id.selectedTheme = i
				break
			}
		}
	} else {
		id.textInput.Focus()
		id.nameInput.Blur()
		id.urlInput.Blur()
		id.tagsInput.Blur()
		id.collectionInput.Blur()
		
		// Set placeholder for text input
		if spec.Placeholder != "" {
			id.textInput.Placeholder = spec.Placeholder
		} else {
			id.textInput.Placeholder = ""
		}
	}
}

func (id *InputDialog) Hide() {
	id.visible = false
	id.confirmed = false
	id.selectedMethod = 0
	id.currentField = 0
	
	// Reset text inputs
	id.textInput.SetValue("")
	id.nameInput.SetValue("")
	id.urlInput.SetValue("")
	id.collectionInput.SetValue("")
	id.selectedFile = ""
	id.useFilePicker = true
	id.selectedTheme = 0
	id.textInput.Blur()
	id.nameInput.Blur()
	id.urlInput.Blur()
	id.collectionInput.Blur()
}

func (id *InputDialog) IsVisible() bool {
	return id.visible
}

func (id *InputDialog) SetInput(input string) {
	id.textInput.SetValue(input)
}

func (id *InputDialog) GetInput() string {
	return id.textInput.Value()
}

func (id *InputDialog) GetResult() (string, string, map[string]interface{}, bool) {
	if id.spec.Type == MethodURLInput {
		// For method/URL input, combine the results
		// Parse tags from comma-separated string
		tagsString := strings.TrimSpace(id.tagsInput.Value())
		var tags []string
		if tagsString != "" {
			for _, tag := range strings.Split(tagsString, ",") {
				trimmed := strings.TrimSpace(tag)
				if trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
		}
		
		result := map[string]interface{}{
			"method": id.methods[id.selectedMethod],
			"name":   id.nameInput.Value(),
			"url":    id.urlInput.Value(),
			"tags":   tags,
		}
		
		// Merge ActionData (contains filePath for edit operations)
		if id.spec.ActionData != nil {
			for k, v := range id.spec.ActionData {
				result[k] = v
			}
		}
		
		return "", id.spec.Action, result, id.confirmed
	} else if id.spec.Type == OpenAPIImportInput {
		// For OpenAPI import, combine URL/file and collection name
		source := ""
		if id.useFilePicker && id.selectedFile != "" {
			source = id.selectedFile
		} else {
			source = id.textInput.Value()
		}
		result := map[string]interface{}{
			"source":     source,
			"collection": id.collectionInput.Value(),
		}
		return "", id.spec.Action, result, id.confirmed
	} else if id.spec.Type == ThemeSelectionInput {
		// For theme selection, return the selected theme name
		if id.selectedTheme < len(id.themes) {
			result := map[string]interface{}{
				"theme": id.themes[id.selectedTheme],
			}
			return "", id.spec.Action, result, id.confirmed
		}
	}
	return id.textInput.Value(), id.spec.Action, id.spec.ActionData, id.confirmed
}

func (id *InputDialog) SwitchField() {
	if id.spec.Type == MethodURLInput {
		id.currentField = (id.currentField + 1) % 4
		// Blur all inputs first
		id.nameInput.Blur()
		id.urlInput.Blur()
		id.tagsInput.Blur()
		
		switch id.currentField {
		case 0:
			// Focus on method selection (no text input)
		case 1:
			// Focus on name input
			id.nameInput.Focus()
		case 2:
			// Focus on URL input
			id.urlInput.Focus()
		case 3:
			// Focus on tags input
			id.tagsInput.Focus()
		}
	} else if id.spec.Type == OpenAPIImportInput {
		// Blur all inputs first
		id.textInput.Blur()
		id.collectionInput.Blur()
		
		if id.useFilePicker {
			// Switch to URL mode - focus on URL input first
			id.useFilePicker = false
			id.textInput.Focus()
			id.currentField = 0 // URL field
		} else {
			// Switch between URL and collection input in URL mode, or back to file picker
			if id.currentField == 0 {
				// Currently on URL, switch to collection
				id.currentField = 1
				id.textInput.Blur()
				id.collectionInput.Focus()
			} else {
				// Currently on collection, switch back to file picker mode
				id.useFilePicker = true
				id.currentField = 0
				id.collectionInput.Blur()
			}
		}
	}
}

func (id *InputDialog) GetCurrentField() int {
	return id.currentField
}

func (id *InputDialog) MoveMethodSelection(direction int) {
	if id.spec.Type == MethodURLInput && id.currentField == 0 {
		id.selectedMethod += direction
		if id.selectedMethod < 0 {
			id.selectedMethod = len(id.methods) - 1
		} else if id.selectedMethod >= len(id.methods) {
			id.selectedMethod = 0
		}
	} else if id.spec.Type == ThemeSelectionInput {
		id.selectedTheme += direction
		if id.selectedTheme < 0 {
			id.selectedTheme = len(id.themes) - 1
		} else if id.selectedTheme >= len(id.themes) {
			id.selectedTheme = 0
		}
	}
}

func (id *InputDialog) SetURLInput(input string) {
	if id.spec.Type == MethodURLInput {
		id.urlInput.SetValue(input)
	}
}

func (id *InputDialog) GetURLInput() string {
	return id.urlInput.Value()
}

func (id *InputDialog) SetNameInput(input string) {
	if id.spec.Type == MethodURLInput {
		id.nameInput.SetValue(input)
	}
}

func (id *InputDialog) GetNameInput() string {
	return id.nameInput.Value()
}

func (id *InputDialog) Confirm() {
	id.confirmed = true
}

// HandleFilePickerUpdate processes file picker updates and returns tea.Cmd
func (id *InputDialog) HandleFilePickerUpdate(msg tea.Msg) tea.Cmd {
	if id.spec.Type == OpenAPIImportInput && id.useFilePicker {
		var cmd tea.Cmd
		id.filePicker, cmd = id.filePicker.Update(msg)
		
		// Check if a file was selected
		if didSelect, path := id.filePicker.DidSelectFile(msg); didSelect {
			id.selectedFile = path
		}
		
		return cmd
	}
	return nil
}

func (id *InputDialog) UpdateTextInputs(msg interface{}) {
	if id.spec.Type == MethodURLInput {
		switch id.currentField {
		case 1:
			// Update name input when it's focused
			id.nameInput, _ = id.nameInput.Update(msg)
		case 2:
			// Update URL input when it's focused
			id.urlInput, _ = id.urlInput.Update(msg)
		case 3:
			// Update tags input when it's focused
			id.tagsInput, _ = id.tagsInput.Update(msg)
		}
	} else if id.spec.Type == OpenAPIImportInput {
		if !id.useFilePicker {
			// URL input mode - update the focused input
			if id.currentField == 0 {
				// URL input
				id.textInput, _ = id.textInput.Update(msg)
			} else if id.currentField == 1 {
				// Collection input
				id.collectionInput, _ = id.collectionInput.Update(msg)
			}
		}
		// Note: File picker is handled separately in HandleFilePickerUpdate
	} else {
		// Update general text input for other types
		id.textInput, _ = id.textInput.Update(msg)
	}
}

func (id *InputDialog) Render(width, height int) string {
	if !id.visible {
		return ""
	}

	// Dialog styles
	dialogStyle := lipgloss.NewStyle().
		Width(width - 20).
		Height(height - 15).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Background(lipgloss.Color("235"))

	// Build content based on input type
	var content strings.Builder

	// Title
	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render(id.spec.Title))
	content.WriteString("\n\n")

	switch id.spec.Type {
	case TextInput:
		// Prompt
		if id.spec.Prompt != "" {
			content.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Render(id.spec.Prompt))
			content.WriteString("\n\n")
		}

		// Use textinput component
		content.WriteString(id.textInput.View())
		content.WriteString("\n\n")

		// Help text
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Enter: Confirm • Esc: Cancel • Ctrl+A: Select All • Ctrl+C/V: Copy/Paste"))

	case ConfirmInput:
		// Confirmation message
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Render(id.spec.Prompt))
		content.WriteString("\n\n")

		// Help text
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("y: Yes • n/Esc: No"))

	case FileInput:
		// File input placeholder (can be extended later)
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Render(id.spec.Prompt))
		content.WriteString("\n\n")

		content.WriteString(id.textInput.View())
		content.WriteString("\n\n")

		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Enter: Select • Esc: Cancel"))

	case MethodURLInput:
		// Method selection
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Render("HTTP Method:"))
		content.WriteString("\n")
		
		methodDisplay := ""
		for i, method := range id.methods {
			if i == id.selectedMethod {
				if id.currentField == 0 {
					methodDisplay += lipgloss.NewStyle().
						Background(lipgloss.Color("62")).
						Foreground(lipgloss.Color("230")).
						Padding(0, 1).
						Render("▶ " + method)
				} else {
					methodDisplay += lipgloss.NewStyle().
						Background(lipgloss.Color("240")).
						Foreground(lipgloss.Color("230")).
						Padding(0, 1).
						Render("  " + method)
				}
			} else {
				methodDisplay += "   " + method
			}
			if i < len(id.methods)-1 {
				methodDisplay += "  "
			}
		}
		content.WriteString(methodDisplay)
		content.WriteString("\n\n")

		// Name input
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Render("Display Name:"))
		content.WriteString("\n")
		
		// Use textinput component for name
		content.WriteString(id.nameInput.View())
		content.WriteString("\n\n")

		// URL input
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Render("URL:"))
		content.WriteString("\n")
		
		// Use textinput component for URL
		content.WriteString(id.urlInput.View())
		content.WriteString("\n\n")

		// Tags input
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Render("Tags (comma-separated):"))
		content.WriteString("\n")
		
		// Use textinput component for tags
		content.WriteString(id.tagsInput.View())
		content.WriteString("\n\n")

		// Help text
		actionText := "Create"
		if id.spec.IsEdit {
			actionText = "Update"
		}
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render(fmt.Sprintf("Tab: Switch fields • ↑↓: Select method • Enter: %s • Esc: Cancel • Ctrl+C/V: Copy/Paste", actionText)))
	
	case OpenAPIImportInput:
		if id.useFilePicker {
			// File picker mode
			content.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Render("Select OpenAPI File:"))
			content.WriteString("\n")
			
			content.WriteString(id.filePicker.View())
			
			if id.selectedFile != "" {
				content.WriteString("\n")
				content.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color("86")).
					Render("Selected: " + id.selectedFile))
			}
			content.WriteString("\n\n")
		} else {
			// URL input mode
			content.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Render("OpenAPI Spec URL:"))
			content.WriteString("\n")
			
			content.WriteString(id.textInput.View())
			content.WriteString("\n\n")
		}

		// Collection name input (only show when not in file picker mode or when file is selected)
		if !id.useFilePicker || id.selectedFile != "" {
			content.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Render("Collection Name (optional):"))
			content.WriteString("\n")
			
			content.WriteString(id.collectionInput.View())
			content.WriteString("\n\n")
		}

		// Help text
		mode := "File Browser"
		if !id.useFilePicker {
			mode = "URL Input"
		}
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render(fmt.Sprintf("Mode: %s • Tab: Switch modes • Enter: Import • Esc: Cancel", mode)))
	
	case ThemeSelectionInput:
		// Theme selection list
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Render("Select Theme:"))
		content.WriteString("\n\n")
		
		for i, theme := range id.themes {
			if i == id.selectedTheme {
				content.WriteString(lipgloss.NewStyle().
					Background(lipgloss.Color("62")).
					Foreground(lipgloss.Color("230")).
					Padding(0, 1).
					Render("▶ " + theme))
			} else {
				content.WriteString("  " + theme)
			}
			if i < len(id.themes)-1 {
				content.WriteString("\n")
			}
		}
		
		content.WriteString("\n\n")
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("↑↓: Navigate • Enter: Apply Theme • Esc: Cancel"))
	}

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
		dialogStyle.Render(content.String()))
}