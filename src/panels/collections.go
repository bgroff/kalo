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
	RequestIndex int // Index into the bruRequests array, -1 for folders
}

func RenderCollections(width, height int, activePanel bool, vp *viewport.Model, focusedStyle, blurredStyle, titleStyle lipgloss.Style) string {
	var style lipgloss.Style
	if activePanel {
		style = focusedStyle
	} else {
		style = blurredStyle
	}

	title := titleStyle.Width(width - 6).Render("Collections")
	
	// Update viewport dimensions
	vp.Width = width - 4  // Account for padding
	vp.Height = height - 4 // Account for title and padding
	
	return style.
		Width(width).
		Height(height).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, vp.View()))
}