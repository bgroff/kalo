package panels

import (
	"fmt"
	"strings"
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

	// Create title with status, timing, and MIME type
	var titleContent string
	if isLoading {
		titleContent = " Response │ ⏳ Loading..."
	} else if lastResponse != nil {
		// Extract MIME type from Content-Type header
		contentType := ""
		if ct, exists := lastResponse.Headers["Content-Type"]; exists {
			// Extract just the MIME type part (before semicolon if present)
			if idx := strings.Index(ct, ";"); idx != -1 {
				contentType = strings.TrimSpace(ct[:idx])
			} else {
				contentType = strings.TrimSpace(ct)
			}
		}
		
		var statusStyle lipgloss.Style
		if statusCode >= 200 && statusCode < 300 {
			statusStyle = statusOkStyle
		} else if statusCode >= 400 {
			statusStyle = lipgloss.NewStyle().Background(lipgloss.Color("196")).Foreground(lipgloss.Color("0")).Padding(0, 1)
		} else {
			statusStyle = lipgloss.NewStyle().Background(lipgloss.Color("214")).Foreground(lipgloss.Color("0")).Padding(0, 1)
		}
		
		statusText := lastResponse.Status
		if statusText == "" {
			statusText = fmt.Sprintf("%d", statusCode)
		}
		
		timing := ""
		if lastResponse.ResponseTime > 0 {
			timing = fmt.Sprintf(" • %v", lastResponse.ResponseTime)
		}
		
		mimeInfo := ""
		if contentType != "" {
			mimeInfo = fmt.Sprintf(" • %s", contentType)
		}
		
		titleContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			" Response │ ",
			statusStyle.Render(statusText),
			timing,
			mimeInfo,
			" ",
		)
	} else {
		titleContent = " Response │ " + statusOkStyle.Render("200 OK") + " • Mock Response"
	}
	
	title := titleStyle.Render(titleContent)
	titleBar := lipgloss.NewStyle().
		Width(width-2).
		Align(lipgloss.Left).
		Render(title)

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

	content := lipgloss.JoinVertical(lipgloss.Left, titleBar, tabsRender, tabContent)

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