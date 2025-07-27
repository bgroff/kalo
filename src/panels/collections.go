package panels

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type CollectionItem struct {
	Name     string
	Type     string
	FilePath string
	IsFolder bool
}

func RenderCollections(width, height int, collections []CollectionItem, selectedReq int, activePanel bool, focusedStyle, blurredStyle, titleStyle lipgloss.Style) string {
	var style lipgloss.Style
	if activePanel {
		style = focusedStyle
	} else {
		style = blurredStyle
	}

	title := titleStyle.Width(width - 6).Render("Collections")
	
	var items []string
	for i, item := range collections {
		if i == selectedReq && activePanel {
			items = append(items, "> "+item.Name)
		} else {
			items = append(items, "  "+item.Name)
		}
	}

	content := strings.Join(items, "\n")
	
	return style.
		Width(width).
		Height(height).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}