package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// CommandPaletteInputHandler handles input for the command palette
type CommandPaletteInputHandler struct{}

// NewCommandPaletteInputHandler creates a new command palette input handler
func NewCommandPaletteInputHandler() *CommandPaletteInputHandler {
	return &CommandPaletteInputHandler{}
}

// HandleInput processes key input for the command palette
func (h *CommandPaletteInputHandler) HandleInput(key tea.KeyMsg, model *model) (*model, tea.Cmd) {
	if !model.commandPalette.IsVisible() {
		return model, nil
	}

	// Use the existing CommandPalette HandleInput method
	_, selectedCmd, handled := model.commandPalette.HandleInput(key)
	if !handled {
		return model, nil
	}

	// If a command was selected, execute it
	if selectedCmd != nil {
		model.commandPalette.Hide()
		return model, model.executeCommand(selectedCmd.Action)
	}

	// If the command palette was closed, just return the model
	if !model.commandPalette.IsVisible() {
		return model, nil
	}

	return model, nil
}

// CanHandleInput returns true if command palette can handle input
func (h *CommandPaletteInputHandler) CanHandleInput(model *model) bool {
	return model.commandPalette.IsVisible()
}

// GetFooterText returns footer text for command palette
func (h *CommandPaletteInputHandler) GetFooterText(model *model) string {
	if !model.commandPalette.IsVisible() {
		return ""
	}
	return "Type command | ↑/↓: navigate | Enter: execute | Esc: cancel"
}

// IsInEditMode returns true if command palette is in edit mode
func (h *CommandPaletteInputHandler) IsInEditMode(model *model) bool {
	return model.commandPalette.IsVisible()
}

