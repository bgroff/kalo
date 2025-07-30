package panels

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type ResponseSection int

const (
	ResponseBodySection ResponseSection = iota
	ResponseHeadersSection
)

type HTTPResponse struct {
	StatusCode   int               `json:"status_code"`
	Status       string            `json:"status"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	ResponseTime time.Duration     `json:"response_time"`
	Error        string            `json:"error,omitempty"`
	IsJSON       bool              `json:"is_json"`
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func GetResponseTabNames() []string {
	return []string{"Response Body", "Response Headers"}
}

func GetResponseTabSection(tabIndex int) ResponseSection {
	switch tabIndex {
	case 0:
		return ResponseBodySection
	case 1:
		return ResponseHeadersSection
	default:
		return ResponseBodySection
	}
}

func renderResponseTabs(tabs []string, activeTab int, width int, focusedStyle, blurredStyle lipgloss.Style, activePanel bool) string {
	var renderedTabs []string
	
	for i, tab := range tabs {
		var tabStyle lipgloss.Style
		if i == activeTab && activePanel {
			tabStyle = focusedStyle.Copy().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderBottom(false)
		} else if i == activeTab {
			tabStyle = blurredStyle.Copy().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderBottom(false)
		} else {
			tabStyle = lipgloss.NewStyle().Padding(0, 1).Faint(true)
		}
		renderedTabs = append(renderedTabs, tabStyle.Render(tab))
	}
	
	tabsContent := lipgloss.JoinHorizontal(lipgloss.Bottom, renderedTabs...)
	
	// Ensure the tabs fit within the available width
	return lipgloss.NewStyle().
		Width(width).
		Render(tabsContent)
}

func RenderResponse(width, height int, activePanel bool, isLoading bool, lastResponse *HTTPResponse, statusCode int, responseCursor ResponseSection, activeTab int, headersViewport, responseViewport *viewport.Model, focusedStyle, blurredStyle, titleStyle, cursorStyle, sectionStyle, statusOkStyle lipgloss.Style, appliedJQFilter string) string {
	var style lipgloss.Style
	if activePanel {
		style = focusedStyle
	} else {
		style = blurredStyle
	}
	
	// Calculate available space for the active tab viewport
	availableHeight := height - 9 // Account for padding, borders, title, tabs
	
	if availableHeight < 5 {
		availableHeight = 5
	}
	
	// Update viewport sizes
	contentWidth := width - 8  // Account for padding and borders
	headersViewport.Width = contentWidth
	headersViewport.Height = availableHeight
	responseViewport.Width = contentWidth
	responseViewport.Height = availableHeight


	// Render tabs (account for panel padding and border)
	tabs := GetResponseTabNames()
	tabsRender := renderResponseTabs(tabs, activeTab, width-4, focusedStyle, blurredStyle, activePanel)

	// Render content for active tab only
	var tabContent string
	currentSection := GetResponseTabSection(activeTab)
	
	if currentSection == ResponseBodySection {
		tabContent = renderResponseBodyContent(responseViewport, activePanel, responseCursor, currentSection, cursorStyle, sectionStyle, appliedJQFilter)
	} else {
		tabContent = renderResponseHeadersContent(headersViewport, activePanel, responseCursor, currentSection, cursorStyle, sectionStyle)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, tabsRender, tabContent)

	return style.
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(content)
}

func renderResponseBodyContent(responseViewport *viewport.Model, activePanel bool, responseCursor ResponseSection, currentSection ResponseSection, cursorStyle, sectionStyle lipgloss.Style, appliedJQFilter string) string {
	bodyScrollInfo := ""
	if responseViewport.TotalLineCount() > 0 {
		scrollPercent := int((float64(responseViewport.YOffset) / float64(max(1, responseViewport.TotalLineCount()-responseViewport.Height))) * 100)
		if scrollPercent > 100 {
			scrollPercent = 100
		}
		bodyScrollInfo = fmt.Sprintf(" [%d%%]", scrollPercent)
		
		// Add jq filter info if one is applied
		if appliedJQFilter != "" {
			bodyScrollInfo += fmt.Sprintf(" (jq: %s)", appliedJQFilter)
		}
	} else if appliedJQFilter != "" {
		// Show jq filter even if no scroll info
		bodyScrollInfo = fmt.Sprintf(" (jq: %s)", appliedJQFilter)
	}
	
	return responseViewport.View()
}

func renderResponseHeadersContent(headersViewport *viewport.Model, activePanel bool, responseCursor ResponseSection, currentSection ResponseSection, cursorStyle, sectionStyle lipgloss.Style) string {
	return headersViewport.View()
}