package panels

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type CollectionItem struct {
	Name         string
	Type         string
	FilePath     string
	IsFolder     bool
	IsTagGroup   bool
	RequestIndex int // Index into the bruRequests array, -1 for folders/tag groups
}

func RenderCollections(width, height int, activePanel bool, vp *viewport.Model, focusedStyle, blurredStyle, titleStyle lipgloss.Style) string {
	var style lipgloss.Style
	if activePanel {
		style = focusedStyle
	} else {
		style = blurredStyle
	}

	// Update viewport dimensions - account for border and inline title
	vp.Width = width - 4  // Account for padding
	vp.Height = height - 3 // Account for padding and inline title
	
	// Create title bar that looks like it's part of the border
	title := titleStyle.Render(" Collections ")
	titleBar := lipgloss.NewStyle().
		Width(width-2).
		Align(lipgloss.Left).
		Render(title)
	
	content := lipgloss.JoinVertical(lipgloss.Left, titleBar, vp.View())
	
	return style.
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(content)
}