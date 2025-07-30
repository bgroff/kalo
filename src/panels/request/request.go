package panels

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderTextCursor renders a solid colored cursor at the specified position in text
func renderTextCursor(text string, cursorPos int, textCursorStyle lipgloss.Style) string {
	if cursorPos < 0 || cursorPos > len(text) {
		return text
	}
	
	if cursorPos == len(text) {
		// Cursor at end - add a space with cursor background
		return text + textCursorStyle.Render(" ")
	} else {
		// Cursor in middle - style the character at cursor position
		before := text[:cursorPos]
		cursorChar := string(text[cursorPos])
		if cursorChar == "" {
			cursorChar = " "
		}
		after := text[cursorPos+1:]
		return before + textCursorStyle.Render(cursorChar) + after
	}
}

type RequestSection int

const (
	QuerySection RequestSection = iota
	BodySection
	HeadersSection
	AuthSection
)

type QueryEditMode int

const (
	QueryViewMode QueryEditMode = iota
	QueryEditKeyMode
	QueryEditValueMode
	QueryAddMode
)

type QueryParameter struct {
	Key   string
	Value string
}

type QueryEditState struct {
	Mode              QueryEditMode
	SelectedIndex     int
	EditingKey        string
	EditingValue      string
	CursorPos         int
	Parameters        []QueryParameter
	OriginalKey       string
	PendingDeletion   int // -1 means no pending deletion
}

type HeaderEditMode int

const (
	HeaderViewMode HeaderEditMode = iota
	HeaderEditKeyMode
	HeaderEditValueMode
	HeaderAddMode
)

type HeaderParameter struct {
	Key   string
	Value string
}

type HeaderEditState struct {
	Mode              HeaderEditMode
	SelectedIndex     int
	EditingKey        string
	EditingValue      string
	CursorPos         int
	Parameters        []HeaderParameter
	OriginalKey       string
	PendingDeletion   int // -1 means no pending deletion
}

type AuthEditMode int

const (
	AuthViewMode AuthEditMode = iota
	AuthTypeSelectMode
	AuthBearerEditMode
	AuthBasicUsernameEditMode
	AuthBasicPasswordEditMode
)

type AuthEditState struct {
	Mode              AuthEditMode
	SelectedField     int    // 0 = type selector, 1 = first field, 2 = second field
	AuthType          string // "none", "bearer", "basic"
	BearerToken       string
	BasicUsername     string
	BasicPassword     string
	CursorPos         int
}

type BodyEditMode int

const (
	BodyViewMode BodyEditMode = iota
	BodyTextEditMode
)

type BodyEditState struct {
	Mode              BodyEditMode
	Content           string
	CursorLine        int
	CursorCol         int
	ScrollOffset      int
	ValidationError   string
	BracketMatches    map[int]int // line -> matching bracket line
}

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
	
	// Edit state for interactive editing
	QueryEditState  *QueryEditState  `json:"-"`
	HeaderEditState *HeaderEditState `json:"-"`
	AuthEditState   *AuthEditState   `json:"-"`
	BodyEditState   *BodyEditState   `json:"-"`
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

// InitializeQueryEditState initializes the query edit state for a request
func (r *BruRequest) InitializeQueryEditState() {
	if r.QueryEditState != nil {
		return
	}
	
	var params []QueryParameter
	for key, value := range r.Query {
		params = append(params, QueryParameter{Key: key, Value: value})
	}
	
	// Sort parameters for consistent display
	sort.Slice(params, func(i, j int) bool {
		return params[i].Key < params[j].Key
	})
	
	r.QueryEditState = &QueryEditState{
		Mode:            QueryViewMode,
		SelectedIndex:   0,
		Parameters:      params,
		PendingDeletion: -1,
	}
}

// SyncQueryToMap updates the Query map from the edit state
func (r *BruRequest) SyncQueryToMap() {
	if r.QueryEditState == nil {
		return
	}
	
	r.Query = make(map[string]string)
	for _, param := range r.QueryEditState.Parameters {
		if param.Key != "" {
			r.Query[param.Key] = param.Value
		}
	}
}

// AddQueryParameter adds a new query parameter
func (r *BruRequest) AddQueryParameter(key, value string) {
	if r.QueryEditState == nil {
		r.InitializeQueryEditState()
	}
	
	r.QueryEditState.Parameters = append(r.QueryEditState.Parameters, QueryParameter{
		Key:   key,
		Value: value,
	})
	r.SyncQueryToMap()
}

// UpdateQueryParameter updates an existing query parameter
func (r *BruRequest) UpdateQueryParameter(index int, key, value string) {
	if r.QueryEditState == nil || index < 0 || index >= len(r.QueryEditState.Parameters) {
		return
	}
	
	r.QueryEditState.Parameters[index].Key = key
	r.QueryEditState.Parameters[index].Value = value
	r.SyncQueryToMap()
}

// DeleteQueryParameter removes a query parameter
func (r *BruRequest) DeleteQueryParameter(index int) {
	if r.QueryEditState == nil || index < 0 || index >= len(r.QueryEditState.Parameters) {
		return
	}
	
	r.QueryEditState.Parameters = append(
		r.QueryEditState.Parameters[:index],
		r.QueryEditState.Parameters[index+1:]...,
	)
	
	// Adjust selected index if necessary
	if r.QueryEditState.SelectedIndex >= len(r.QueryEditState.Parameters) && len(r.QueryEditState.Parameters) > 0 {
		r.QueryEditState.SelectedIndex = len(r.QueryEditState.Parameters) - 1
	}
	
	r.SyncQueryToMap()
}


// InitializeHeaderEditState initializes the header edit state for a request
func (r *BruRequest) InitializeHeaderEditState() {
	if r.HeaderEditState != nil {
		return
	}
	
	var params []HeaderParameter
	for key, value := range r.Headers {
		params = append(params, HeaderParameter{Key: key, Value: value})
	}
	
	// Sort parameters for consistent display
	sort.Slice(params, func(i, j int) bool {
		return params[i].Key < params[j].Key
	})
	
	r.HeaderEditState = &HeaderEditState{
		Mode:            HeaderViewMode,
		SelectedIndex:   0,
		Parameters:      params,
		PendingDeletion: -1,
	}
}

// SyncHeadersToMap updates the Headers map from the edit state
func (r *BruRequest) SyncHeadersToMap() {
	if r.HeaderEditState == nil {
		return
	}
	
	r.Headers = make(map[string]string)
	for _, param := range r.HeaderEditState.Parameters {
		if param.Key != "" {
			r.Headers[param.Key] = param.Value
		}
	}
}

// AddHeader adds a new header
func (r *BruRequest) AddHeader(key, value string) {
	if r.HeaderEditState == nil {
		r.InitializeHeaderEditState()
	}
	
	r.HeaderEditState.Parameters = append(r.HeaderEditState.Parameters, HeaderParameter{
		Key:   key,
		Value: value,
	})
	r.SyncHeadersToMap()
}

// UpdateHeader updates an existing header
func (r *BruRequest) UpdateHeader(index int, key, value string) {
	if r.HeaderEditState == nil || index < 0 || index >= len(r.HeaderEditState.Parameters) {
		return
	}
	
	r.HeaderEditState.Parameters[index].Key = key
	r.HeaderEditState.Parameters[index].Value = value
	r.SyncHeadersToMap()
}

// DeleteHeader removes a header
func (r *BruRequest) DeleteHeader(index int) {
	if r.HeaderEditState == nil || index < 0 || index >= len(r.HeaderEditState.Parameters) {
		return
	}
	
	r.HeaderEditState.Parameters = append(
		r.HeaderEditState.Parameters[:index],
		r.HeaderEditState.Parameters[index+1:]...,
	)
	
	// Adjust selected index if necessary
	if r.HeaderEditState.SelectedIndex >= len(r.HeaderEditState.Parameters) && len(r.HeaderEditState.Parameters) > 0 {
		r.HeaderEditState.SelectedIndex = len(r.HeaderEditState.Parameters) - 1
	}
	
	r.SyncHeadersToMap()
}

// InitializeAuthEditState initializes the auth edit state for a request
func (r *BruRequest) InitializeAuthEditState() {
	if r.AuthEditState != nil {
		return
	}
	
	// Determine current auth type and extract values
	authType := "none"
	bearerToken := ""
	basicUsername := ""
	basicPassword := ""
	
	if r.Auth.Type == "bearer" {
		authType = "bearer"
		if token, exists := r.Auth.Values["token"]; exists {
			bearerToken = token
		}
	} else if r.Auth.Type == "basic" {
		authType = "basic"
		if username, exists := r.Auth.Values["username"]; exists {
			basicUsername = username
		}
		if password, exists := r.Auth.Values["password"]; exists {
			basicPassword = password
		}
	}
	
	r.AuthEditState = &AuthEditState{
		Mode:          AuthViewMode,
		SelectedField: 0,
		AuthType:      authType,
		BearerToken:   bearerToken,
		BasicUsername: basicUsername,
		BasicPassword: basicPassword,
		CursorPos:     0,
	}
}

// SyncAuthToRequest updates the BruAuth from the edit state
func (r *BruRequest) SyncAuthToRequest() {
	if r.AuthEditState == nil {
		return
	}
	
	switch r.AuthEditState.AuthType {
	case "none":
		r.Auth.Type = ""
		r.Auth.Values = nil
	case "bearer":
		r.Auth.Type = "bearer"
		if r.Auth.Values == nil {
			r.Auth.Values = make(map[string]string)
		}
		r.Auth.Values["token"] = r.AuthEditState.BearerToken
	case "basic":
		r.Auth.Type = "basic"
		if r.Auth.Values == nil {
			r.Auth.Values = make(map[string]string)
		}
		r.Auth.Values["username"] = r.AuthEditState.BasicUsername
		r.Auth.Values["password"] = r.AuthEditState.BasicPassword
	}
}

// InitializeBodyEditState initializes the body edit state for a request
func (r *BruRequest) InitializeBodyEditState() {
	if r.BodyEditState != nil {
		return
	}
	
	content := r.Body.Data
	if content == "" {
		content = ""
	}
	
	r.BodyEditState = &BodyEditState{
		Mode:            BodyViewMode,
		Content:         content,
		CursorLine:      0,
		CursorCol:       0,
		ScrollOffset:    0,
		ValidationError: "",
		BracketMatches:  make(map[int]int),
	}
	
	// Validate JSON if body type is JSON
	if r.Body.Type == "json" {
		r.ValidateBodyJSON()
	}
}

// SyncBodyToRequest updates the Body from the edit state
func (r *BruRequest) SyncBodyToRequest() {
	if r.BodyEditState == nil {
		return
	}
	
	r.Body.Data = r.BodyEditState.Content
	
	// Validate JSON if body type is JSON
	if r.Body.Type == "json" {
		r.ValidateBodyJSON()
	}
}

// ValidateBodyJSON validates the JSON content and updates validation error
func (r *BruRequest) ValidateBodyJSON() {
	if r.BodyEditState == nil {
		return
	}
	
	content := strings.TrimSpace(r.BodyEditState.Content)
	if content == "" {
		r.BodyEditState.ValidationError = ""
		return
	}
	
	var jsonData interface{}
	if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
		r.BodyEditState.ValidationError = err.Error()
	} else {
		r.BodyEditState.ValidationError = ""
	}
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

func RenderRequest(width, height int, currentReq *BruRequest, activePanel bool, requestCursor RequestSection, activeTab int, focusedStyle, blurredStyle, titleStyle, cursorStyle, methodStyle, urlStyle, sectionStyle, textCursorStyle lipgloss.Style) string {
	var style lipgloss.Style
	if activePanel {
		style = focusedStyle
	} else {
		style = blurredStyle
	}

	if currentReq == nil {
		content := "No request selected"
		return style.
			Width(width).
			Height(height).
			Padding(0, 1).
			Render(content)
	}

	// Render tabs (account for panel padding and border)
	tabs := GetRequestTabNames()
	tabsRender := renderTabs(tabs, activeTab, width-4, focusedStyle, blurredStyle, activePanel)

	// Render content for active tab only
	var tabContent string
	currentSection := GetRequestTabSection(activeTab)
	
	switch currentSection {
	case QuerySection:
		tabContent = renderQueryContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle, textCursorStyle)
	case BodySection:
		tabContent = renderBodyContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle)
	case HeadersSection:
		tabContent = renderHeadersContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle, textCursorStyle)
	case AuthSection:
		tabContent = renderAuthContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, tabsRender, tabContent)

	return style.
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(content)
}

func renderQueryContent(currentReq *BruRequest, activePanel bool, requestCursor RequestSection, currentSection RequestSection, cursorStyle, sectionStyle, textCursorStyle lipgloss.Style) string {
	// Initialize edit state if needed
	currentReq.InitializeQueryEditState()
	editState := currentReq.QueryEditState
	
	if len(editState.Parameters) == 0 {
		helpText := "  No query parameters"
		if activePanel && requestCursor == currentSection {
			helpText += "\n\n  Press 'a' to add a parameter"
		}
		return helpText
	}

	var lines []string
	
	// Add help text at the top when in query section
	if activePanel && requestCursor == currentSection {
		switch editState.Mode {
		case QueryViewMode:
			lines = append(lines, "  ↑↓: Navigate • Enter/i: Edit • a: Add • d: Delete • Esc: Cancel")
		case QueryEditKeyMode:
			lines = append(lines, "  Tab: Edit value • Enter: Save • Esc: Cancel")
		case QueryEditValueMode:
			lines = append(lines, "  Shift+Tab: Edit key • Enter: Save • Esc: Cancel")
		case QueryAddMode:
			lines = append(lines, "  Tab: Switch key/value • Enter: Save • Esc: Cancel")
		}
		lines = append(lines, "")
	}

	// Calculate maximum key length for alignment
	maxKeyLength := 0
	for _, param := range editState.Parameters {
		if len(param.Key) > maxKeyLength {
			maxKeyLength = len(param.Key)
		}
	}
	if maxKeyLength < 8 { // Minimum width
		maxKeyLength = 8
	}
	
	// Render each parameter
	for i, param := range editState.Parameters {
		var line string
		isSelected := activePanel && requestCursor == currentSection && i == editState.SelectedIndex
		
		// Handle different editing modes
		switch {
		case editState.Mode == QueryEditKeyMode && isSelected:
			// Editing key
			keyWithCursor := renderTextCursor(editState.EditingKey, editState.CursorPos, textCursorStyle)
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, keyWithCursor)
			line = fmt.Sprintf("  %s: %s", paddedKey, param.Value)
			line = cursorStyle.Render(line)
			
		case editState.Mode == QueryEditValueMode && isSelected:
			// Editing value
			valueWithCursor := renderTextCursor(editState.EditingValue, editState.CursorPos, textCursorStyle)
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, param.Key)
			line = fmt.Sprintf("  %s: %s", paddedKey, valueWithCursor)
			line = cursorStyle.Render(line)
			
		case editState.PendingDeletion == i:
			// Show deletion confirmation
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, param.Key)
			line = fmt.Sprintf("  %s: %s [DELETE? y/n]", paddedKey, param.Value)
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(line)
			
		case isSelected:
			// Selected but not editing
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, param.Key)
			line = fmt.Sprintf("  %s: %s", paddedKey, param.Value)
			line = cursorStyle.Render(line)
			
		default:
			// Normal display
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, param.Key)
			line = fmt.Sprintf("  %s: %s", paddedKey, param.Value)
		}
		
		lines = append(lines, line)
	}
	
	// Show add parameter input if in add mode
	if editState.Mode == QueryAddMode && activePanel && requestCursor == currentSection {
		lines = append(lines, "")
		keyWithCursor := renderTextCursor(editState.EditingKey, editState.CursorPos, textCursorStyle)
		paddedKey := fmt.Sprintf("%-*s", maxKeyLength, keyWithCursor)
		line := fmt.Sprintf("  %s: %s", paddedKey, editState.EditingValue)
		lines = append(lines, cursorStyle.Render(line))
	}
	
	return strings.Join(lines, "\n")
}

func renderBodyContent(currentReq *BruRequest, activePanel bool, requestCursor RequestSection, currentSection RequestSection, cursorStyle, sectionStyle lipgloss.Style) string {
	// Initialize edit state if needed
	currentReq.InitializeBodyEditState()
	editState := currentReq.BodyEditState
	
	var lines []string
	
	// Add help text at the top when in body section
	if activePanel && requestCursor == currentSection {
		switch editState.Mode {
		case BodyViewMode:
			lines = append(lines, "  Enter/i: Edit • Esc: Cancel")
		case BodyTextEditMode:
			lines = append(lines, "  Ctrl+S: Save • Esc: Cancel • Arrow keys: Navigate")
		}
		lines = append(lines, "")
	}
	
	// Show body type
	lines = append(lines, fmt.Sprintf("  Type: %s", currentReq.Body.Type))
	
	// Show validation error if exists
	if editState.ValidationError != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
		lines = append(lines, "")
		lines = append(lines, errorStyle.Render("  ✗ JSON Error: "+editState.ValidationError))
	}
	
	lines = append(lines, "")
	
	if editState.Content == "" {
		helpText := "  No request body"
		if activePanel && requestCursor == currentSection {
			helpText += "\n\n  Press Enter or 'i' to start editing"
		}
		lines = append(lines, helpText)
		return strings.Join(lines, "\n")
	}
	
	// Render content based on mode
	if editState.Mode == BodyTextEditMode && activePanel && requestCursor == currentSection {
		// Edit mode: show content with cursor
		contentLines := renderBodyEditMode(editState, currentReq.Body.Type, cursorStyle)
		lines = append(lines, contentLines...)
	} else {
		// View mode: show formatted content
		contentLines := renderBodyViewMode(editState, currentReq.Body.Type)
		lines = append(lines, contentLines...)
	}
	
	return strings.Join(lines, "\n")
}

// renderBodyViewMode renders the body in view mode with syntax highlighting
func renderBodyViewMode(editState *BodyEditState, bodyType string) []string {
	content := editState.Content
	
	if bodyType == "json" {
		return renderJSONContent(content, false, 0, 0)
	}
	
	// For non-JSON content, just show as plain text with indentation
	contentLines := strings.Split(content, "\n")
	var lines []string
	for _, line := range contentLines {
		lines = append(lines, "  "+line)
	}
	return lines
}

// renderBodyEditMode renders the body in edit mode with cursor
func renderBodyEditMode(editState *BodyEditState, bodyType string, cursorStyle lipgloss.Style) []string {
	content := editState.Content
	
	if bodyType == "json" {
		return renderJSONContent(content, true, editState.CursorLine, editState.CursorCol)
	}
	
	// For non-JSON content, show with cursor
	contentLines := strings.Split(content, "\n")
	var lines []string
	
	for i, line := range contentLines {
		prefix := "  "
		if i == editState.CursorLine {
			// Show cursor on this line
			if editState.CursorCol <= len(line) {
				before := line[:editState.CursorCol]
				var cursorChar string
				var after string
				
				if editState.CursorCol < len(line) {
					cursorChar = string(line[editState.CursorCol])
					after = line[editState.CursorCol+1:]
				} else {
					cursorChar = " "
					after = ""
				}
				
				cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("240"))
				lineWithCursor := before + cursorStyle.Render(cursorChar) + after
				lines = append(lines, prefix+lineWithCursor)
			} else {
				cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("240"))
				lines = append(lines, prefix+line+cursorStyle.Render(" "))
			}
		} else {
			lines = append(lines, prefix+line)
		}
	}
	
	return lines
}

// renderJSONContent renders JSON with syntax highlighting
func renderJSONContent(content string, showCursor bool, cursorLine, cursorCol int) []string {
	// Define styles for JSON syntax highlighting
	stringStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))     // Green
	numberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))     // Yellow
	boolStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))       // Blue
	nullStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))       // Gray
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))        // Cyan
	bracketStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))    // Magenta
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("240"))
	
	contentLines := strings.Split(content, "\n")
	var lines []string
	
	for i, line := range contentLines {
		prefix := "  "
		highlightedLine := highlightJSONLine(line, stringStyle, numberStyle, boolStyle, nullStyle, keyStyle, bracketStyle)
		
		if showCursor && i == cursorLine {
			// Add cursor to this line
			if cursorCol <= len(line) {
				// Split the highlighted line at cursor position
				before := highlightJSONLine(line[:cursorCol], stringStyle, numberStyle, boolStyle, nullStyle, keyStyle, bracketStyle)
				var cursorChar string
				var after string
				
				if cursorCol < len(line) {
					cursorChar = string(line[cursorCol])
					after = highlightJSONLine(line[cursorCol+1:], stringStyle, numberStyle, boolStyle, nullStyle, keyStyle, bracketStyle)
				} else {
					cursorChar = " "
					after = ""
				}
				
				lineWithCursor := before + cursorStyle.Render(cursorChar) + after
				lines = append(lines, prefix+lineWithCursor)
			} else {
				lines = append(lines, prefix+highlightedLine+cursorStyle.Render(" "))
			}
		} else {
			lines = append(lines, prefix+highlightedLine)
		}
	}
	
	return lines
}

// highlightJSONLine applies syntax highlighting to a single line of JSON
func highlightJSONLine(line string, stringStyle, numberStyle, boolStyle, nullStyle, keyStyle, bracketStyle lipgloss.Style) string {
	if strings.TrimSpace(line) == "" {
		return line
	}
	
	result := ""
	inString := false
	isKey := false
	escaped := false
	
	for i, char := range line {
		switch char {
		case '"':
			if !escaped {
				inString = !inString
				if inString {
					// Check if this might be a key (look ahead for :)
					remaining := line[i:]
					colonIndex := strings.Index(remaining, ":")
					nextQuoteIndex := strings.Index(remaining[1:], "\"")
					if colonIndex != -1 && (nextQuoteIndex == -1 || colonIndex < nextQuoteIndex+1) {
						isKey = true
					}
				}
			}
			if inString && isKey {
				result += keyStyle.Render(string(char))
			} else if inString {
				result += stringStyle.Render(string(char))
			} else {
				result += string(char)
			}
			escaped = false
		case '\\':
			if inString {
				if isKey {
					result += keyStyle.Render(string(char))
				} else {
					result += stringStyle.Render(string(char))
				}
				escaped = !escaped
			} else {
				result += string(char)
				escaped = false
			}
		case '{', '}', '[', ']':
			if !inString {
				result += bracketStyle.Render(string(char))
			} else {
				if isKey {
					result += keyStyle.Render(string(char))
				} else {
					result += stringStyle.Render(string(char))
				}
			}
			escaped = false
		case ':':
			if !inString {
				result += string(char)
				isKey = false
			} else {
				if isKey {
					result += keyStyle.Render(string(char))
				} else {
					result += stringStyle.Render(string(char))
				}
			}
			escaped = false
		default:
			if inString {
				if isKey {
					result += keyStyle.Render(string(char))
				} else {
					result += stringStyle.Render(string(char))
				}
			} else {
				// Check for numbers, booleans, null
				remaining := line[i:]
				if char >= '0' && char <= '9' || char == '-' || char == '.' {
					// Might be a number, find the end
					numEnd := i
					for j := i; j < len(line); j++ {
						c := line[j]
						if (c >= '0' && c <= '9') || c == '.' || c == '-' || c == 'e' || c == 'E' || c == '+' {
							numEnd = j
						} else {
							break
						}
					}
					if numEnd > i {
						number := line[i:numEnd+1]
						result += numberStyle.Render(number)
						// Skip ahead
						for j := i + 1; j <= numEnd; j++ {
							if j < len(line) {
								char = rune(line[j])
							}
						}
						continue
					}
				} else if strings.HasPrefix(remaining, "true") {
					result += boolStyle.Render("true")
					// Skip ahead 3 more characters
					for j := 0; j < 3 && i+j+1 < len(line); j++ {
						char = rune(line[i+j+1])
					}
					continue
				} else if strings.HasPrefix(remaining, "false") {
					result += boolStyle.Render("false")
					// Skip ahead 4 more characters
					for j := 0; j < 4 && i+j+1 < len(line); j++ {
						char = rune(line[i+j+1])
					}
					continue
				} else if strings.HasPrefix(remaining, "null") {
					result += nullStyle.Render("null")
					// Skip ahead 3 more characters
					for j := 0; j < 3 && i+j+1 < len(line); j++ {
						char = rune(line[i+j+1])
					}
					continue
				}
				
				result += string(char)
			}
			escaped = false
		}
	}
	
	return result
}

func renderHeadersContent(currentReq *BruRequest, activePanel bool, requestCursor RequestSection, currentSection RequestSection, cursorStyle, sectionStyle, textCursorStyle lipgloss.Style) string {
	// Initialize edit state if needed
	currentReq.InitializeHeaderEditState()
	editState := currentReq.HeaderEditState
	
	if len(editState.Parameters) == 0 {
		helpText := "  No headers"
		if activePanel && requestCursor == currentSection {
			helpText += "\n\n  Press 'a' to add a header"
		}
		return helpText
	}

	var lines []string
	
	// Add help text at the top when in headers section
	if activePanel && requestCursor == currentSection {
		switch editState.Mode {
		case HeaderViewMode:
			lines = append(lines, "  ↑↓: Navigate • Enter/i: Edit • a: Add • d: Delete • Esc: Cancel")
		case HeaderEditKeyMode:
			lines = append(lines, "  Tab: Edit value • Enter: Save • Esc: Cancel")
		case HeaderEditValueMode:
			lines = append(lines, "  Shift+Tab: Edit key • Enter: Save • Esc: Cancel")
		case HeaderAddMode:
			lines = append(lines, "  Tab: Switch key/value • Enter: Save • Esc: Cancel")
		}
		lines = append(lines, "")
	}

	// Calculate maximum key length for alignment
	maxKeyLength := 0
	for _, param := range editState.Parameters {
		if len(param.Key) > maxKeyLength {
			maxKeyLength = len(param.Key)
		}
	}
	if maxKeyLength < 8 { // Minimum width
		maxKeyLength = 8
	}
	
	// Render each header
	for i, param := range editState.Parameters {
		var line string
		isSelected := activePanel && requestCursor == currentSection && i == editState.SelectedIndex
		
		// Handle different editing modes
		switch {
		case editState.Mode == HeaderEditKeyMode && isSelected:
			// Editing key
			keyWithCursor := renderTextCursor(editState.EditingKey, editState.CursorPos, textCursorStyle)
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, keyWithCursor)
			line = fmt.Sprintf("  %s: %s", paddedKey, param.Value)
			line = cursorStyle.Render(line)
			
		case editState.Mode == HeaderEditValueMode && isSelected:
			// Editing value
			valueWithCursor := renderTextCursor(editState.EditingValue, editState.CursorPos, textCursorStyle)
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, param.Key)
			line = fmt.Sprintf("  %s: %s", paddedKey, valueWithCursor)
			line = cursorStyle.Render(line)
			
		case editState.PendingDeletion == i:
			// Show deletion confirmation
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, param.Key)
			line = fmt.Sprintf("  %s: %s [DELETE? y/n]", paddedKey, param.Value)
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(line)
			
		case isSelected:
			// Selected but not editing
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, param.Key)
			line = fmt.Sprintf("  %s: %s", paddedKey, param.Value)
			line = cursorStyle.Render(line)
			
		default:
			// Normal display
			paddedKey := fmt.Sprintf("%-*s", maxKeyLength, param.Key)
			line = fmt.Sprintf("  %s: %s", paddedKey, param.Value)
		}
		
		lines = append(lines, line)
	}
	
	// Show add header input if in add mode
	if editState.Mode == HeaderAddMode && activePanel && requestCursor == currentSection {
		lines = append(lines, "")
		keyWithCursor := renderTextCursor(editState.EditingKey, editState.CursorPos, textCursorStyle)
		paddedKey := fmt.Sprintf("%-*s", maxKeyLength, keyWithCursor)
		line := fmt.Sprintf("  %s: %s", paddedKey, editState.EditingValue)
		lines = append(lines, cursorStyle.Render(line))
	}
	
	return strings.Join(lines, "\n")
}

func renderAuthContent(currentReq *BruRequest, activePanel bool, requestCursor RequestSection, currentSection RequestSection, cursorStyle, sectionStyle lipgloss.Style) string {
	// Initialize edit state if needed
	currentReq.InitializeAuthEditState()
	editState := currentReq.AuthEditState
	
	var lines []string
	
	// Add help text at the top when in auth section
	if activePanel && requestCursor == currentSection {
		switch editState.Mode {
		case AuthViewMode:
			lines = append(lines, "  ↑↓: Navigate • Enter/i: Edit • Esc: Cancel")
		case AuthTypeSelectMode:
			lines = append(lines, "  ↑↓: Select type • Enter: Confirm • Esc: Cancel")
		case AuthBearerEditMode:
			lines = append(lines, "  Type token • Enter: Save • Esc: Cancel")
		case AuthBasicUsernameEditMode, AuthBasicPasswordEditMode:
			lines = append(lines, "  Type credentials • Tab: Next field • Enter: Save • Esc: Cancel")
		}
		lines = append(lines, "")
	}
	
	// Render auth type selector
	typeText := "None"
	switch editState.AuthType {
	case "bearer":
		typeText = "Bearer Token"
	case "basic":
		typeText = "Basic Auth"
	}
	
	var typeLine string
	if editState.Mode == AuthTypeSelectMode && activePanel && requestCursor == currentSection {
		// Show dropdown options
		options := []string{"None", "Bearer Token", "Basic Auth"}
		selectedIndex := 0
		switch editState.AuthType {
		case "bearer":
			selectedIndex = 1
		case "basic":
			selectedIndex = 2
		}
		
		typeLine = "  Type: " + typeText + " ▼"
		lines = append(lines, cursorStyle.Render(typeLine))
		
		// Show dropdown options
		for i, option := range options {
			prefix := "    "
			if i == selectedIndex {
				prefix = "  > "
				lines = append(lines, cursorStyle.Render(prefix + option))
			} else {
				lines = append(lines, prefix + option)
			}
		}
	} else {
		typeLine = "  Type: " + typeText
		isTypeSelected := activePanel && requestCursor == currentSection && editState.SelectedField == 0 && editState.Mode == AuthViewMode
		if isTypeSelected {
			lines = append(lines, cursorStyle.Render(typeLine))
		} else {
			lines = append(lines, typeLine)
		}
	}
	
	// Render fields based on auth type
	switch editState.AuthType {
	case "bearer":
		lines = append(lines, "")
		
		var tokenLine string
		if editState.Mode == AuthBearerEditMode && activePanel && requestCursor == currentSection {
			// Show token with cursor
			tokenWithCursor := renderTextCursor(editState.BearerToken, editState.CursorPos, lipgloss.NewStyle().Background(lipgloss.Color("240")))
			tokenLine = "  Token: " + tokenWithCursor
			lines = append(lines, cursorStyle.Render(tokenLine))
		} else {
			tokenLine = "  Token: " + editState.BearerToken
			isTokenSelected := activePanel && requestCursor == currentSection && editState.SelectedField == 1 && editState.Mode == AuthViewMode
			if isTokenSelected {
				lines = append(lines, cursorStyle.Render(tokenLine))
			} else {
				lines = append(lines, tokenLine)
			}
		}
		
	case "basic":
		lines = append(lines, "")
		
		// Username field
		var userLine string
		if editState.Mode == AuthBasicUsernameEditMode && activePanel && requestCursor == currentSection {
			usernameWithCursor := renderTextCursor(editState.BasicUsername, editState.CursorPos, lipgloss.NewStyle().Background(lipgloss.Color("240")))
			userLine = "  Username: " + usernameWithCursor
			lines = append(lines, cursorStyle.Render(userLine))
		} else {
			userLine = "  Username: " + editState.BasicUsername
			isUsernameSelected := activePanel && requestCursor == currentSection && editState.SelectedField == 1 && editState.Mode == AuthViewMode
			if isUsernameSelected {
				lines = append(lines, cursorStyle.Render(userLine))
			} else {
				lines = append(lines, userLine)
			}
		}
		
		// Password field
		var passLine string
		if editState.Mode == AuthBasicPasswordEditMode && activePanel && requestCursor == currentSection {
			// Show password with cursor (masked)
			maskedPassword := strings.Repeat("*", len(editState.BasicPassword))
			passwordWithCursor := renderTextCursor(maskedPassword, editState.CursorPos, lipgloss.NewStyle().Background(lipgloss.Color("240")))
			passLine = "  Password: " + passwordWithCursor
			lines = append(lines, cursorStyle.Render(passLine))
		} else {
			// Show masked password
			maskedPassword := strings.Repeat("*", len(editState.BasicPassword))
			passLine = "  Password: " + maskedPassword
			isPasswordSelected := activePanel && requestCursor == currentSection && editState.SelectedField == 2 && editState.Mode == AuthViewMode
			if isPasswordSelected {
				lines = append(lines, cursorStyle.Render(passLine))
			} else {
				lines = append(lines, passLine)
			}
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