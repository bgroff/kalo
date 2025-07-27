package panels

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type RequestSection int

const (
	URLSection RequestSection = iota
	HeadersSection
	QuerySection
	AuthSection
	BodySection
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

func RenderRequest(width, height int, currentReq *BruRequest, activePanel bool, requestCursor RequestSection, focusedStyle, blurredStyle, titleStyle, cursorStyle, methodStyle, urlStyle, sectionStyle lipgloss.Style) string {
	var style lipgloss.Style
	if activePanel {
		style = focusedStyle
	} else {
		style = blurredStyle
	}

	title := titleStyle.Width(width - 6).Render("Request")
	
	if currentReq == nil {
		content := "No request selected"
		return style.
			Width(width).
			Height(height).
			Padding(1, 2).
			Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
	}

	var sections []string

	urlCursor := ""
	if activePanel && requestCursor == URLSection {
		urlCursor = cursorStyle.Render("► ")
	}
	methodAndUrl := lipgloss.JoinHorizontal(
		lipgloss.Left,
		urlCursor,
		methodStyle.Render(currentReq.HTTP.Method),
		" ",
		urlStyle.Render(currentReq.HTTP.URL),
	)
	sections = append(sections, methodAndUrl)

	if len(currentReq.Headers) > 0 {
		headersCursor := ""
		if activePanel && requestCursor == HeadersSection {
			headersCursor = cursorStyle.Render("► ")
		}
		headers := headersCursor + sectionStyle.Render("Headers:")
		sections = append(sections, headers)
		
		var headersContent strings.Builder
		// Sort headers alphabetically
		var keys []string
		for key := range currentReq.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		
		for _, key := range keys {
			headersContent.WriteString(fmt.Sprintf("  %s: %s\n", key, currentReq.Headers[key]))
		}
		sections = append(sections, strings.TrimSpace(headersContent.String()))
	}

	if len(currentReq.Query) > 0 {
		queryCursor := ""
		if activePanel && requestCursor == QuerySection {
			queryCursor = cursorStyle.Render("► ")
		}
		queryHeader := queryCursor + sectionStyle.Render("Query Parameters:")
		sections = append(sections, queryHeader)
		
		var queryContent strings.Builder
		// Sort query parameters alphabetically
		var keys []string
		for key := range currentReq.Query {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		
		for _, key := range keys {
			queryContent.WriteString(fmt.Sprintf("  %s: %s\n", key, currentReq.Query[key]))
		}
		sections = append(sections, strings.TrimSpace(queryContent.String()))
	}

	if currentReq.Auth.Type != "" {
		authCursor := ""
		if activePanel && requestCursor == AuthSection {
			authCursor = cursorStyle.Render("► ")
		}
		authHeader := authCursor + sectionStyle.Render(fmt.Sprintf("Auth (%s):", currentReq.Auth.Type))
		sections = append(sections, authHeader)
		
		var authContent strings.Builder
		for key, value := range currentReq.Auth.Values {
			authContent.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
		sections = append(sections, strings.TrimSpace(authContent.String()))
	}

	if currentReq.Body.Type != "" && currentReq.Body.Data != "" {
		bodyCursor := ""
		if activePanel && requestCursor == BodySection {
			bodyCursor = cursorStyle.Render("► ")
		}
		bodyHeader := bodyCursor + sectionStyle.Render(fmt.Sprintf("Body (%s):", currentReq.Body.Type))
		sections = append(sections, bodyHeader)
		sections = append(sections, "  "+currentReq.Body.Data)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return style.
		Width(width).
		Height(height).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

func GetMaxRequestSection(currentReq *BruRequest) RequestSection {
	if currentReq == nil {
		return URLSection
	}

	maxSection := URLSection

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