package main

import tea "github.com/charmbracelet/bubbletea"

// PanelInputHandler defines the interface that all panel input handlers must implement
type PanelInputHandler interface {
	// HandleInput processes a key input for this panel
	HandleInput(key tea.KeyMsg, model *model) (*model, tea.Cmd)
	
	// CanHandleInput returns true if this panel can currently handle input
	CanHandleInput(model *model) bool
	
	// GetFooterText returns the footer text to display when this panel is active
	GetFooterText(model *model) string
	
	// IsInEditMode returns true if the panel is currently in an edit mode
	IsInEditMode(model *model) bool
}