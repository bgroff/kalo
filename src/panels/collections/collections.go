package panels

import (
	"strings"

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
	IsExpanded   bool // Whether folder/tag is expanded
	IsVisible    bool // Whether item should be visible (considering parent expansion)
}

// UpdateCollectionsViewport updates the viewport content for collections
func UpdateCollectionsViewport(collections []CollectionItem, selectedReq int, vp *viewport.Model, textCursorStyle lipgloss.Style) {
	var items []string
	visibleIndex := 0
	selectedVisibleIndex := -1
	
	for i, item := range collections {
		if !item.IsVisible {
			continue
		}
		
		// Track which visible item corresponds to the selected index
		if i == selectedReq {
			selectedVisibleIndex = visibleIndex
		}
		
		// Add expand/collapse indicator for folders and tag groups
		displayName := item.Name
		if item.IsFolder || item.IsTagGroup {
			// Extract existing indentation and content
			indentLevel := ""
			content := displayName
			
			// Find where the actual content starts (after spaces)
			for i, char := range displayName {
				if char != ' ' {
					indentLevel = displayName[:i]
					content = displayName[i:]
					break
				}
			}
			
			// Add +/- at the same indentation level
			if item.IsExpanded {
				displayName = indentLevel + "- " + content
			} else {
				displayName = indentLevel + "+ " + content
			}
		}
		
		if i == selectedReq {
			// Use solid colored bar spanning the whole row for selected item
			items = append(items, textCursorStyle.Render("  "+displayName))
		} else {
			items = append(items, "  "+displayName)
		}
		visibleIndex++
	}
	
	content := strings.Join(items, "\n")
	vp.SetContent(content)
	
	// Ensure the selected item is visible (use visible index for scrolling)
	if selectedVisibleIndex >= 0 {
		viewportHeight := vp.Height
		
		// If selected item is below the visible area, scroll down
		if selectedVisibleIndex >= vp.YOffset+viewportHeight {
			vp.SetYOffset(selectedVisibleIndex - viewportHeight + 1)
		}
		// If selected item is above the visible area, scroll up
		if selectedVisibleIndex < vp.YOffset {
			vp.SetYOffset(selectedVisibleIndex)
		}
	}
}

// ToggleExpansion toggles the expansion state of a folder or tag group
func ToggleExpansion(collections []CollectionItem, index int) {
	if index < 0 || index >= len(collections) {
		return
	}
	
	item := &collections[index]
	if !item.IsFolder && !item.IsTagGroup {
		return
	}
	
	// Toggle expansion state
	item.IsExpanded = !item.IsExpanded
}

// UpdateVisibility updates the visibility state of all collection items
func UpdateVisibility(collections []CollectionItem) []CollectionItem {
	for i := range collections {
		item := &collections[i]
		
		if item.IsFolder {
			// Folders are always visible
			item.IsVisible = true
		} else if item.IsTagGroup {
			// Tag groups are visible if their parent folder is expanded (or if they're root tags)
			item.IsVisible = true
			
			// Find parent folder
			for j := i - 1; j >= 0; j-- {
				if collections[j].IsFolder {
					item.IsVisible = collections[j].IsExpanded
					break
				}
			}
		} else {
			// Requests are visible if their parent tag group is expanded
			item.IsVisible = false
			
			// Find parent tag group
			for j := i - 1; j >= 0; j-- {
				parentItem := &collections[j]
				if parentItem.IsTagGroup {
					item.IsVisible = parentItem.IsExpanded
					break
				}
				if parentItem.IsFolder {
					// If we hit a folder before finding a tag group, this is a direct child of folder
					item.IsVisible = parentItem.IsExpanded
					break
				}
			}
		}
	}
	
	return collections
}

func RenderCollections(width, height int, activePanel bool, vp *viewport.Model, focusedStyle, blurredStyle, titleStyle lipgloss.Style) string {
	var style lipgloss.Style
	if activePanel {
		style = focusedStyle
	} else {
		style = blurredStyle
	}

	// Update viewport dimensions - no internal title now
	vp.Width = width - 4  // Account for padding
	vp.Height = height - 2 // Account for padding only
	
	return style.
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(vp.View())
}