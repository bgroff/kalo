package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Command struct {
	Name        string
	Description string
	Action      string
}

type CommandPalette struct {
	visible         bool
	textInput       textinput.Model
	cursor          int
	commands        []Command
	filteredCommands []Command
}

func NewCommandPalette() *CommandPalette {
	commands := []Command{
		{Name: "Create Collection", Description: "Create a new collection", Action: "create_collection"},
		{Name: "New Request", Description: "Create a new request file", Action: "new_request"},
		{Name: "Edit Request", Description: "Edit the current request", Action: "edit_request"},
		{Name: "Import OpenAPI", Description: "Import OpenAPI 3.x specification", Action: "import_openapi"},
		{Name: "Import Collection", Description: "Import Bruno collection", Action: "import_collection"},
		{Name: "Settings", Description: "Open application settings", Action: "settings"},
	}

	ti := textinput.New()
	ti.Focus()
	ti.Placeholder = "Search commands..."
	ti.Width = 50

	cp := &CommandPalette{
		visible:   false,
		textInput: ti,
		commands:  commands,
		cursor:    0,
	}
	cp.updateFiltered()
	return cp
}

func (cp *CommandPalette) Show() {
	cp.visible = true
	cp.textInput.SetValue("")
	cp.textInput.Focus()
	cp.cursor = 0
	cp.updateFiltered()
}

func (cp *CommandPalette) Hide() {
	cp.visible = false
	cp.textInput.SetValue("")
	cp.textInput.Blur()
	cp.cursor = 0
}

func (cp *CommandPalette) IsVisible() bool {
	return cp.visible
}

func (cp *CommandPalette) SetInput(input string) {
	cp.textInput.SetValue(input)
	cp.cursor = 0
	cp.updateFiltered()
}

func (cp *CommandPalette) GetInput() string {
	return cp.textInput.Value()
}

func (cp *CommandPalette) UpdateTextInput(msg interface{}) {
	var cmd tea.Cmd
	cp.textInput, cmd = cp.textInput.Update(msg)
	if cmd != nil {
		// Handle any commands if needed
	}
	cp.updateFiltered()
}

// HandleInput processes keyboard input for the command palette
// Returns: (shouldHide bool, selectedCommand *Command, handled bool)
func (cp *CommandPalette) HandleInput(msg tea.KeyMsg) (bool, *Command, bool) {
	if !cp.visible {
		return false, nil, false
	}

	switch msg.String() {
	case "esc":
		cp.Hide()
		return true, nil, true
	case "enter":
		if cmd := cp.GetSelectedCommand(); cmd != nil {
			cp.Hide()
			return true, cmd, true
		}
		return false, nil, true
	case "up":
		cp.MoveCursor(-1)
		return false, nil, true
	case "down":
		cp.MoveCursor(1)
		return false, nil, true
	default:
		// Let textinput component handle all other input (including copy/paste)
		cp.UpdateTextInput(msg)
		return false, nil, true
	}
}

func (cp *CommandPalette) MoveCursor(direction int) {
	if len(cp.filteredCommands) == 0 {
		return
	}
	
	cp.cursor += direction
	if cp.cursor < 0 {
		cp.cursor = len(cp.filteredCommands) - 1
	} else if cp.cursor >= len(cp.filteredCommands) {
		cp.cursor = 0
	}
}

func (cp *CommandPalette) GetSelectedCommand() *Command {
	if len(cp.filteredCommands) == 0 || cp.cursor >= len(cp.filteredCommands) {
		return nil
	}
	return &cp.filteredCommands[cp.cursor]
}


func (cp *CommandPalette) updateFiltered() {
	cp.filteredCommands = []Command{}
	searchTerm := strings.ToLower(cp.textInput.Value())
	
	for _, cmd := range cp.commands {
		if strings.Contains(strings.ToLower(cmd.Name), searchTerm) ||
		   strings.Contains(strings.ToLower(cmd.Description), searchTerm) {
			cp.filteredCommands = append(cp.filteredCommands, cmd)
		}
	}
}

func (cp *CommandPalette) Render(width, height int) string {
	if !cp.visible {
		return ""
	}

	// Command palette styles
	paletteStyle := lipgloss.NewStyle().
		Width(width - 20).
		Height(height - 10).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Background(lipgloss.Color("235"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1).
		Width(width - 24)

	normalStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(width - 24)

	// Build content
	var content strings.Builder
	
	// Title
	content.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("Command Palette"))
	content.WriteString("\n\n")
	
	// Input field using textinput component
	content.WriteString(cp.textInput.View())
	content.WriteString("\n\n")
	
	// Commands list
	if len(cp.filteredCommands) == 0 {
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No commands found"))
	} else {
		for i, cmd := range cp.filteredCommands {
			var line string
			if i == cp.cursor {
				line = selectedStyle.Render("▶ " + cmd.Name + " - " + cmd.Description)
			} else {
				line = normalStyle.Render("  " + cmd.Name + " - " + cmd.Description)
			}
			content.WriteString(line)
			if i < len(cp.filteredCommands)-1 {
				content.WriteString("\n")
			}
		}
	}
	
	// Help text
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("↑↓: Navigate • Enter: Execute • Esc: Close"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, 
		paletteStyle.Render(content.String()))
}