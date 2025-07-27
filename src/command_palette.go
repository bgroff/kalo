package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Command struct {
	Name        string
	Description string
	Action      string
}

type CommandPalette struct {
	visible         bool
	input           string
	cursor          int
	commands        []Command
	filteredCommands []Command
}

func NewCommandPalette() *CommandPalette {
	commands := []Command{
		{Name: "Create Collection", Description: "Create a new collection", Action: "create_collection"},
		{Name: "New Request", Description: "Create a new request file", Action: "new_request"},
		{Name: "Import Collection", Description: "Import Bruno collection", Action: "import_collection"},
		{Name: "Settings", Description: "Open application settings", Action: "settings"},
	}

	cp := &CommandPalette{
		visible:  false,
		commands: commands,
		cursor:   0,
	}
	cp.updateFiltered()
	return cp
}

func (cp *CommandPalette) Show() {
	cp.visible = true
	cp.input = ""
	cp.cursor = 0
	cp.updateFiltered()
}

func (cp *CommandPalette) Hide() {
	cp.visible = false
	cp.input = ""
	cp.cursor = 0
}

func (cp *CommandPalette) IsVisible() bool {
	return cp.visible
}

func (cp *CommandPalette) SetInput(input string) {
	cp.input = input
	cp.cursor = 0
	cp.updateFiltered()
}

func (cp *CommandPalette) GetInput() string {
	return cp.input
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
	searchTerm := strings.ToLower(cp.input)
	
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

	inputStyle := lipgloss.NewStyle().
		Width(width - 24).
		Padding(0, 1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Background(lipgloss.Color("0"))

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
	
	// Input field
	inputDisplay := cp.input + "█" // Simple cursor
	content.WriteString(inputStyle.Render(inputDisplay))
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