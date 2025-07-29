package panels

import (
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
	TagsSection
)

type QueryEditMode int

const (
	QueryViewMode QueryEditMode = iota
	QueryEditKeyMode
	QueryEditValueMode
	QueryAddMode
)

type TagEditMode int

const (
	TagViewMode TagEditMode = iota
	TagEditingMode
	TagAddMode
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

type TagEditState struct {
	Mode            TagEditMode
	SelectedIndex   int
	EditingTag      string
	CursorPos       int
	Tags            []string
	PendingDeletion int // -1 means no pending deletion
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
	TagEditState    *TagEditState    `json:"-"`
	HeaderEditState *HeaderEditState `json:"-"`
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

// InitializeTagEditState initializes the tag edit state for a request
func (r *BruRequest) InitializeTagEditState() {
	if r.TagEditState != nil {
		return
	}
	
	// Create a copy of tags for editing
	tags := make([]string, len(r.Tags))
	copy(tags, r.Tags)
	
	r.TagEditState = &TagEditState{
		Mode:            TagViewMode,
		SelectedIndex:   0,
		Tags:            tags,
		PendingDeletion: -1,
	}
}

// SyncTagsToSlice updates the Tags slice from the edit state
func (r *BruRequest) SyncTagsToSlice() {
	if r.TagEditState == nil {
		return
	}
	
	r.Tags = make([]string, len(r.TagEditState.Tags))
	copy(r.Tags, r.TagEditState.Tags)
}

// AddTag adds a new tag
func (r *BruRequest) AddTag(tag string) {
	if r.TagEditState == nil {
		r.InitializeTagEditState()
	}
	
	// Check if tag already exists to avoid duplicates
	for _, existingTag := range r.TagEditState.Tags {
		if existingTag == tag {
			return
		}
	}
	
	r.TagEditState.Tags = append(r.TagEditState.Tags, tag)
	r.SyncTagsToSlice()
}

// UpdateTag updates an existing tag
func (r *BruRequest) UpdateTag(index int, newTag string) {
	if r.TagEditState == nil || index < 0 || index >= len(r.TagEditState.Tags) {
		return
	}
	
	// Check if new tag already exists (and is not the current one being edited)
	for i, existingTag := range r.TagEditState.Tags {
		if i != index && existingTag == newTag {
			return
		}
	}
	
	r.TagEditState.Tags[index] = newTag
	r.SyncTagsToSlice()
}

// DeleteTag removes a tag
func (r *BruRequest) DeleteTag(index int) {
	if r.TagEditState == nil || index < 0 || index >= len(r.TagEditState.Tags) {
		return
	}
	
	r.TagEditState.Tags = append(
		r.TagEditState.Tags[:index],
		r.TagEditState.Tags[index+1:]...,
	)
	
	// Adjust selected index if necessary
	if r.TagEditState.SelectedIndex >= len(r.TagEditState.Tags) && len(r.TagEditState.Tags) > 0 {
		r.TagEditState.SelectedIndex = len(r.TagEditState.Tags) - 1
	}
	
	r.SyncTagsToSlice()
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

func GetRequestTabNames() []string {
	return []string{"Query Parameters", "Request Body", "Headers", "Authorization", "Tags"}
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
	case 4:
		return TagsSection
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
		tabContent = renderQueryContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle, textCursorStyle)
	case BodySection:
		tabContent = renderBodyContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle)
	case HeadersSection:
		tabContent = renderHeadersContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle, textCursorStyle)
	case AuthSection:
		tabContent = renderAuthContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle)
	case TagsSection:
		tabContent = renderTagsContent(currentReq, activePanel, requestCursor, currentSection, cursorStyle, sectionStyle, textCursorStyle)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, titleBar, tabsRender, tabContent)

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
	if currentReq.Body.Type == "" || currentReq.Body.Data == "" {
		return "  No request body"
	}

	return fmt.Sprintf("  Type: %s\n\n%s", currentReq.Body.Type, currentReq.Body.Data)
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

func renderTagsContent(currentReq *BruRequest, activePanel bool, requestCursor RequestSection, currentSection RequestSection, cursorStyle, sectionStyle, textCursorStyle lipgloss.Style) string {
	// Initialize edit state if needed
	currentReq.InitializeTagEditState()
	editState := currentReq.TagEditState
	
	if len(editState.Tags) == 0 {
		helpText := "  No tags"
		if activePanel && requestCursor == currentSection {
			helpText += "\n\n  Press 'a' to add a tag"
		}
		return helpText
	}

	var lines []string
	
	// Add help text at the top when in tags section
	if activePanel && requestCursor == currentSection {
		switch editState.Mode {
		case TagViewMode:
			lines = append(lines, "  ↑↓: Navigate • Enter/i: Edit • a: Add • d: Delete • Esc: Cancel")
		case TagEditingMode:
			lines = append(lines, "  Enter: Save • Esc: Cancel")
		case TagAddMode:
			lines = append(lines, "  Enter: Save • Esc: Cancel")
		}
		lines = append(lines, "")
	}
	
	// Render each tag
	for i, tag := range editState.Tags {
		var line string
		isSelected := activePanel && requestCursor == currentSection && i == editState.SelectedIndex
		
		// Handle different editing modes
		switch {
		case editState.Mode == TagEditingMode && isSelected:
			// Editing tag
			tagWithCursor := renderTextCursor(editState.EditingTag, editState.CursorPos, textCursorStyle)
			line = fmt.Sprintf("  %s", tagWithCursor)
			line = cursorStyle.Render(line)
			
		case editState.PendingDeletion == i:
			// Show deletion confirmation
			line = fmt.Sprintf("  %s [DELETE? y/n]", tag)
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(line)
			
		case isSelected:
			// Selected but not editing
			line = fmt.Sprintf("  %s", tag)
			line = cursorStyle.Render(line)
			
		default:
			// Normal display
			line = fmt.Sprintf("  %s", tag)
		}
		
		lines = append(lines, line)
	}
	
	// Show add tag input if in add mode
	if editState.Mode == TagAddMode && activePanel && requestCursor == currentSection {
		lines = append(lines, "")
		tagWithCursor := renderTextCursor(editState.EditingTag, editState.CursorPos, textCursorStyle)
		line := fmt.Sprintf("  %s", tagWithCursor)
		lines = append(lines, cursorStyle.Render(line))
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
	if len(currentReq.Tags) > 0 {
		maxSection = TagsSection
	}

	return maxSection
}