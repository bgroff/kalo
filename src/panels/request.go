package panels

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type RequestSection int

const (
	QuerySection RequestSection = iota
	BodySection
	HeadersSection
	AuthSection
)

type BruRequest struct {
	Meta    BruMeta           `json:"meta"`
	HTTP    BruHTTP           `json:"http"`
	Headers map[string]string `json:"headers,omitempty"`
	Query   map[string]string `json:"query,omitempty"`
	Body    BruBody           `json:"body,omitempty"`
	Auth    BruAuth           `json:"auth,omitempty"`
	Vars    map[string]string `json:"vars,omitempty"`
	Tests   string            `json:"tests,omitempty"`
	Docs    string            `json:"docs,omitempty"`
	Tags    []string          `json:"tags,omitempty"`
}

type BruMeta struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Seq  int    `json:"seq"`
}

type BruHTTP struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

type BruBody struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type BruAuth struct {
	Type   string            `json:"type"`
	Values map[string]string `json:"values,omitempty"`
}

func GetRequestTabNames() []string {
	return []string{"Query Parameters", "Request Body", "Headers", "Authorization"}
}

func GetRequestTabSection(tabIndex int) RequestSection {
	switch tabIndex {
	case 0:
		return QuerySection
	case 1:
		return BodySection
	case 2:
		return HeadersSection
	case 3:
		return AuthSection
	default:
		return QuerySection
	}
}

func renderTabs(tabs []string, activeTab int, width int, focusedStyle, blurredStyle lipgloss.Style, activePanel bool) string {
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

func RenderRequest(width, height int, currentReq *BruRequest, activePanel bool, requestCursor RequestSection, activeTab int, focusedStyle, blurredStyle, titleStyle, cursorStyle, methodStyle, urlStyle, sectionStyle lipgloss.Style) string {
	var style lipgloss.Style
	if activePanel {
		style = focusedStyle
	} else {
		style = blurredStyle
	}

	if currentReq == nil {
		title := titleStyle.Render(" Request ")
		titleBar := lipgloss.NewStyle().
			Width(width-2).
			Align(lipgloss.Left).
			Render(title)
		
		content := lipgloss.JoinVertical(lipgloss.Left, titleBar, "No request selected")
		return style.
			Width(width).
			Height(height).
			Padding(0, 1).
			Render(content)
	}

	// Create title with method and URL
	titleContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		" Request ",
		"│ ",
		methodStyle.Render(currentReq.HTTP.Method),
		" ",
		urlStyle.Render(currentReq.HTTP.URL),
		" ",
	)
	title := titleStyle.Render(titleContent)
	titleBar := lipgloss.NewStyle().
		Width(width-2).
		Align(lipgloss.Left).
		Render(title)

	// Render tabs (account for panel padding and border)
	tabs := GetRequestTabNames()
	tabsRender := renderTabs(tabs, activeTab, width-4, focusedStyle, blurredStyle, activePanel)

	// Render content for active tab only
	var tabContent string
	currentSection := GetRequestTabSection(activeTab)
	
	switch currentSection {
	case QuerySection:
		tabContent = renderQueryContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle)
	case BodySection:
		tabContent = renderBodyContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle)
	case HeadersSection:
		tabContent = renderHeadersContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle)
	case AuthSection:
		tabContent = renderAuthContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, titleBar, tabsRender, tabContent)

	return style.
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(content)
}

func renderQueryContent(currentReq *BruRequest, activePanel bool, requestCursor RequestSection, currentSection RequestSection, cursorStyle, sectionStyle lipgloss.Style) string {
	if len(currentReq.Query) == 0 {
		return "  No query parameters"
	}

	// Sort query parameters alphabetically
	var keys []string
	maxKeyLength := 0
	for key := range currentReq.Query {
		keys = append(keys, key)
		if len(key) > maxKeyLength {
			maxKeyLength = len(key)
		}
	}
	sort.Strings(keys)
	
	var lines []string
	for _, key := range keys {
		// Align the colons by padding the key to the maximum key length
		paddedKey := fmt.Sprintf("%-*s", maxKeyLength, key)
		lines = append(lines, fmt.Sprintf("  %s: %s", paddedKey, currentReq.Query[key]))
	}
	return strings.Join(lines, "\n")
}

func renderBodyContent(currentReq *BruRequest, activePanel bool, requestCursor RequestSection, currentSection RequestSection, cursorStyle, sectionStyle lipgloss.Style) string {
	if currentReq.Body.Type == "" || currentReq.Body.Data == "" {
		return "  No request body"
	}

	return fmt.Sprintf("  Type: %s\n\n%s", currentReq.Body.Type, currentReq.Body.Data)
}

func renderHeadersContent(currentReq *BruRequest, activePanel bool, requestCursor RequestSection, currentSection RequestSection, cursorStyle, sectionStyle lipgloss.Style) string {
	if len(currentReq.Headers) == 0 {
		return "  No headers"
	}

	// Sort headers alphabetically
	var keys []string
	maxKeyLength := 0
	for key := range currentReq.Headers {
		keys = append(keys, key)
		if len(key) > maxKeyLength {
			maxKeyLength = len(key)
		}
	}
	sort.Strings(keys)
	
	var lines []string
	for _, key := range keys {
		// Align the colons by padding the key to the maximum key length
		paddedKey := fmt.Sprintf("%-*s", maxKeyLength, key)
		lines = append(lines, fmt.Sprintf("  %s: %s", paddedKey, currentReq.Headers[key]))
	}
	return strings.Join(lines, "\n")
}

func renderAuthContent(currentReq *BruRequest, activePanel bool, requestCursor RequestSection, currentSection RequestSection, cursorStyle, sectionStyle lipgloss.Style) string {
	if currentReq.Auth.Type == "" {
		return "  No authorization"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("  Type: %s", currentReq.Auth.Type))
	
	if len(currentReq.Auth.Values) > 0 {
		lines = append(lines, "") // Empty line separator
		
		// Find the maximum key length for alignment
		maxKeyLength := 0
		var keys []string
		for key := range currentReq.Auth.Values {
			keys = append(keys, key)
			if len(key) > maxKeyLength {
				maxKeyLength = len(key)
			}
		}
		sort.Strings(keys)
		
		for _, key := range keys {
			// Align the colons by padding the key to the maximum key length
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, key)
			lines = append(lines, fmt.Sprintf("  %s: %s", paddedKey, currentReq.Auth.Values[key]))
		}
	}
	
	return strings.Join(lines, "\n")
}

func GetMaxRequestSection(currentReq *BruRequest) RequestSection {
	if currentReq == nil {
		return QuerySection
	}

	maxSection := QuerySection

	if len(currentReq.Headers) > 0 {
		maxSection = HeadersSection
	}
	if len(currentReq.Query) > 0 {
		maxSection = QuerySection
	}
	if currentReq.Auth.Type != "" {
		maxSection = AuthSection
	}
	if currentReq.Body.Type != "" && currentReq.Body.Data != "" {
		maxSection = BodySection
	}

	return maxSection
}