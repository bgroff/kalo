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
	ResponseHeadersSection ResponseSection = iota
	ResponseBodySection
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

func RenderResponse(width, height int, activePanel bool, isLoading bool, lastResponse *HTTPResponse, statusCode int, responseCursor ResponseSection, headersViewport, responseViewport *viewport.Model, focusedStyle, blurredStyle, titleStyle, cursorStyle, sectionStyle, statusOkStyle lipgloss.Style, appliedJQFilter string) string {
	var style lipgloss.Style
	if activePanel {
		style = focusedStyle
	} else {
		style = blurredStyle
	}
	
	// Calculate available space for both viewports
	availableHeight := height - 6 // Account for padding, borders, status line
	headersHeight := availableHeight / 3  // 1/3 for headers
	bodyHeight := availableHeight - headersHeight // 2/3 for body
	
	if headersHeight < 3 {
		headersHeight = 3
	}
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	
	// Update viewport sizes
	contentWidth := width - 8  // Account for padding and borders
	headersViewport.Width = contentWidth
	headersViewport.Height = headersHeight
	responseViewport.Width = contentWidth
	responseViewport.Height = bodyHeight

	// Headers section with cursor and scroll indicator
	headersCursor := ""
	if activePanel && responseCursor == ResponseHeadersSection {
		headersCursor = cursorStyle.Render("► ")
	}
	
	headersScrollInfo := ""
	if headersViewport.TotalLineCount() > 0 {
		scrollPercent := int((float64(headersViewport.YOffset) / float64(max(1, headersViewport.TotalLineCount()-headersViewport.Height))) * 100)
		if scrollPercent > 100 {
			scrollPercent = 100
		}
		headersScrollInfo = fmt.Sprintf(" [%d%%]", scrollPercent)
	}
	
	headersTitle := headersCursor + sectionStyle.Render("Response Headers:") + headersScrollInfo

	// Body section with cursor and scroll indicator
	bodyCursor := ""
	if activePanel && responseCursor == ResponseBodySection {
		bodyCursor = cursorStyle.Render("► ")
	}
	
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
	
	bodyTitle := bodyCursor + sectionStyle.Render("Response Body:") + bodyScrollInfo

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

	responseContent := lipgloss.JoinVertical(
		lipgloss.Left,
		headersTitle,
		headersViewport.View(),
		bodyTitle,
		responseViewport.View(),
	)
	
	content := lipgloss.JoinVertical(lipgloss.Left, titleBar, responseContent)

	return style.
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(content)
}