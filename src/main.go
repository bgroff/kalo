package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/itchyny/gojq"

	"kalo/src/panels"
)

type panel int

const (
	collectionsPanel panel = iota
	requestPanel
	responsePanel
)

type httpResponseMsg struct {
	response *panels.HTTPResponse
	err      error
}

type importCompleteMsg struct {
	success bool
	err     error
}

type jqFilterMsg struct {
	result string
	err    error
}

type FilterType string

const (
	JQFilter          FilterType = "jq"
	CollectionsFilter FilterType = "collections"
)

type filterMsg struct {
	filterType FilterType
	result     string
	err        error
}

type model struct {
	width            int
	height           int
	activePanel      panel
	collections      []panels.CollectionItem
	selectedReq      int
	bruRequests      []*panels.BruRequest
	currentReq       *panels.BruRequest
	requestCursor    panels.RequestSection
	response         string
	statusCode       int
	httpClient       *HTTPClient
	lastResponse     *panels.HTTPResponse
	isLoading        bool
	collectionsViewport viewport.Model
	responseViewport viewport.Model
	headersViewport  viewport.Model
	responseCursor   panels.ResponseSection
	originalResponse string // Store original response for jq filtering
	commandPalette   *CommandPalette
	inputDialog      *InputDialog
	filterMode       bool
	filterType       FilterType
	filterInput      string
	originalCollections []panels.CollectionItem // Store original collections for filtering
	lastJQFilter     string // Remember last jq filter
	lastCollectionsFilter string // Remember last collections filter
	appliedJQFilter  string // Currently applied jq filter for display
	jqSuggestions    []string // Available jq suggestions
	selectedSuggestion int   // Currently selected suggestion index
	showSuggestions  bool    // Whether to show the suggestions popup
	filterCursorPos  int     // Cursor position in filter input
}

var (
	focusedStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	blurredStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)

	methodStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("46")).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1).
			Bold(true)

	urlStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))

	statusOkStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("46")).
			Foreground(lipgloss.Color("0")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	sectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))
)

func getCollectionsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	collectionsDir := filepath.Join(homeDir, ".kalo", "collections")
	
	// Create the directory if it doesn't exist
	err = os.MkdirAll(collectionsDir, 0755)
	if err != nil {
		return "", err
	}
	
	return collectionsDir, nil
}

func initialModel() model {
	collectionsVP := viewport.New(30, 20)
	collectionsVP.SetContent("Loading collections...")

	responseVP := viewport.New(30, 10)
	responseVP.SetContent(`{
  "message": "Select a request to see response"
}`)

	headersVP := viewport.New(30, 5)
	headersVP.SetContent("No headers available")

	m := model{
		activePanel:         collectionsPanel,
		selectedReq:         0,
		requestCursor:       panels.URLSection,
		responseCursor:      panels.ResponseHeadersSection,
		statusCode:          200,
		httpClient:          NewHTTPClient(),
		collectionsViewport: collectionsVP,
		responseViewport:    responseVP,
		headersViewport:     headersVP,
		commandPalette:      NewCommandPalette(),
		inputDialog:         NewInputDialog(),
		response: `{
  "message": "Select a request to see response"
}`,
	}
	m.loadBruFiles()
	return m
}

func (m *model) loadBruFiles() {
	m.collections = []panels.CollectionItem{}
	m.bruRequests = []*panels.BruRequest{}

	collectionsDir, err := getCollectionsDir()
	if err != nil {
		return
	}

	// Check if collections directory exists and has content
	if _, err := os.Stat(collectionsDir); os.IsNotExist(err) {
		return
	}

	// First, collect all directories (collections)
	dirEntries, err := os.ReadDir(collectionsDir)
	if err != nil {
		return
	}

	// Group requests by collection and then by tag
	collections := make(map[string]map[string][]*panels.BruRequest)
	requestPaths := make(map[*panels.BruRequest]string)

	// Add collection folders first and load their requests
	for _, entry := range dirEntries {
		if entry.IsDir() {
			collectionName := entry.Name()
			collectionPath := filepath.Join(collectionsDir, collectionName)
			
			collections[collectionName] = make(map[string][]*panels.BruRequest)

			// Load .bru files from this collection
			collectionFiles, err := os.ReadDir(collectionPath)
			if err != nil {
				continue
			}

			for _, file := range collectionFiles {
				if strings.HasSuffix(file.Name(), ".bru") {
					bruPath := filepath.Join(collectionPath, file.Name())
					
					bruFile, err := os.Open(bruPath)
					if err != nil {
						continue
					}

					parser := NewBruParser(bruFile)
					request, err := parser.Parse()
					bruFile.Close()
					if err != nil {
						continue
					}

					m.bruRequests = append(m.bruRequests, request)
					requestPaths[request] = bruPath

					// Group by tags, or use "untagged" if no tags
					if len(request.Tags) == 0 {
						collections[collectionName]["untagged"] = append(collections[collectionName]["untagged"], request)
					} else {
						for _, tag := range request.Tags {
							collections[collectionName][tag] = append(collections[collectionName][tag], request)
						}
					}
				}
			}
		}
	}

	// Add any standalone .bru files in the root collections directory
	rootRequests := make(map[string][]*panels.BruRequest)
	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".bru") {
			bruPath := filepath.Join(collectionsDir, entry.Name())
			
			bruFile, err := os.Open(bruPath)
			if err != nil {
				continue
			}

			parser := NewBruParser(bruFile)
			request, err := parser.Parse()
			bruFile.Close()
			if err != nil {
				continue
			}

			m.bruRequests = append(m.bruRequests, request)
			requestPaths[request] = bruPath

			// Group by tags, or use "untagged" if no tags
			if len(request.Tags) == 0 {
				rootRequests["untagged"] = append(rootRequests["untagged"], request)
			} else {
				for _, tag := range request.Tags {
					rootRequests[tag] = append(rootRequests[tag], request)
				}
			}
		}
	}

	// Build the display list with tag grouping
	requestIndexMap := make(map[*panels.BruRequest]int)
	for i, req := range m.bruRequests {
		requestIndexMap[req] = i
	}

	// Add collection folders with tag grouping
	collectionNames := make([]string, 0, len(collections))
	for name := range collections {
		collectionNames = append(collectionNames, name)
	}
	sort.Strings(collectionNames)

	for _, collectionName := range collectionNames {
		tags := collections[collectionName]
		
		// Only add collection folder if it has requests
		hasRequests := false
		for _, requests := range tags {
			if len(requests) > 0 {
				hasRequests = true
				break
			}
		}
		
		if !hasRequests {
			continue
		}

		// Add collection folder header
		m.collections = append(m.collections, panels.CollectionItem{
			Name:         "📁 " + collectionName,
			Type:         "folder",
			FilePath:     filepath.Join(collectionsDir, collectionName),
			IsFolder:     true,
			IsTagGroup:   false,
			RequestIndex: -1,
			IsExpanded:   false, // Collapsed by default
			IsVisible:    true,  // Folders are always visible
		})

		// Get sorted tag names
		tagNames := make([]string, 0, len(tags))
		for tagName := range tags {
			if len(tags[tagName]) > 0 { // Only include tags that have requests
				tagNames = append(tagNames, tagName)
			}
		}
		sort.Strings(tagNames)

		// Add tag groups and requests
		for _, tagName := range tagNames {
			requests := tags[tagName]

			// Add tag group header (unless it's "untagged" and only one tag group)
			if len(tagNames) > 1 || tagName != "untagged" {
				tagDisplayName := "🏷️ " + tagName
				if tagName == "untagged" {
					tagDisplayName = "📄 untagged"
				}
				m.collections = append(m.collections, panels.CollectionItem{
					Name:         "    " + tagDisplayName,
					Type:         "tag",
					FilePath:     "",
					IsFolder:     false,
					IsTagGroup:   true,
					RequestIndex: -1,
					IsExpanded:   false, // Collapsed by default
					IsVisible:    false, // Hidden when parent folder is collapsed
				})
			}

			// Add requests under this tag
			for _, request := range requests {
				methodColor := getMethodColor(request.HTTP.Method)
				indentLevel := "        "
				if len(tagNames) == 1 && tagName == "untagged" {
					indentLevel = "    " // Less indentation if no tag groups
				}
				displayName := fmt.Sprintf("%s%s %s", indentLevel, methodColor, request.Meta.Name)

				m.collections = append(m.collections, panels.CollectionItem{
					Name:         displayName,
					Type:         "request",
					FilePath:     requestPaths[request],
					IsFolder:     false,
					IsTagGroup:   false,
					RequestIndex: requestIndexMap[request],
					IsExpanded:   false, // Not applicable for requests
					IsVisible:    false, // Hidden when parent tag/folder is collapsed
				})
			}
		}
	}

	// Add root requests with tag grouping
	if len(rootRequests) > 0 {
		// Get sorted tag names
		tagNames := make([]string, 0, len(rootRequests))
		for tagName := range rootRequests {
			tagNames = append(tagNames, tagName)
		}
		sort.Strings(tagNames)

		// Add tag groups and requests
		for _, tagName := range tagNames {
			requests := rootRequests[tagName]
			if len(requests) == 0 {
				continue
			}

			// Add tag group header (unless it's "untagged" and only one tag group)
			if len(tagNames) > 1 || tagName != "untagged" {
				tagDisplayName := "🏷️ " + tagName
				if tagName == "untagged" {
					tagDisplayName = "📄 untagged"
				}
				m.collections = append(m.collections, panels.CollectionItem{
					Name:         tagDisplayName,
					Type:         "tag",
					FilePath:     "",
					IsFolder:     false,
					IsTagGroup:   true,
					RequestIndex: -1,
					IsExpanded:   false, // Collapsed by default
					IsVisible:    true,  // Root tags are visible
				})
			}

			// Add requests under this tag
			for _, request := range requests {
				methodColor := getMethodColor(request.HTTP.Method)
				indentLevel := "    "
				if len(tagNames) == 1 && tagName == "untagged" {
					indentLevel = "  " // Less indentation if no tag groups
				}
				displayName := fmt.Sprintf("%s%s %s", indentLevel, methodColor, request.Meta.Name)

				m.collections = append(m.collections, panels.CollectionItem{
					Name:         displayName,
					Type:         "request",
					FilePath:     requestPaths[request],
					IsFolder:     false,
					IsTagGroup:   false,
					RequestIndex: requestIndexMap[request],
					IsExpanded:   false, // Not applicable for requests
					IsVisible:    false, // Hidden when parent tag/folder is collapsed
				})
			}
		}
	}

	if len(m.bruRequests) > 0 {
		m.currentReq = m.bruRequests[0]
	}
	
	// Update visibility based on expansion state
	m.updateVisibility()
	
	// Ensure selected item is visible
	if m.selectedReq >= 0 && m.selectedReq < len(m.collections) && !m.collections[m.selectedReq].IsVisible {
		// Find first visible item
		for i := 0; i < len(m.collections); i++ {
			if m.collections[i].IsVisible {
				m.selectedReq = i
				break
			}
		}
	}
	
	// Update collections viewport content
	m.updateCollectionsViewport()
}

func (m *model) toggleExpansion(index int) {
	if index < 0 || index >= len(m.collections) {
		return
	}
	
	item := &m.collections[index]
	if !item.IsFolder && !item.IsTagGroup {
		return
	}
	
	// Toggle expansion state
	item.IsExpanded = !item.IsExpanded
	
	// Update visibility of child items
	m.updateVisibility()
}

func (m *model) updateVisibility() {
	for i := range m.collections {
		item := &m.collections[i]
		
		if item.IsFolder {
			// Folders are always visible
			item.IsVisible = true
		} else if item.IsTagGroup {
			// Tag groups are visible if their parent folder is expanded (or if they're root tags)
			item.IsVisible = true
			
			// Find parent folder
			for j := i - 1; j >= 0; j-- {
				if m.collections[j].IsFolder {
					item.IsVisible = m.collections[j].IsExpanded
					break
				}
			}
		} else {
			// Requests are visible if their parent tag group is expanded
			item.IsVisible = false
			
			// Find parent tag group
			for j := i - 1; j >= 0; j-- {
				parentItem := &m.collections[j]
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
}

func (m *model) updateCollectionsViewport() {
	var items []string
	visibleIndex := 0
	selectedVisibleIndex := -1
	
	for i, item := range m.collections {
		if !item.IsVisible {
			continue
		}
		
		// Track which visible item corresponds to the selected index
		if i == m.selectedReq {
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
		
		if i == m.selectedReq {
			items = append(items, "> "+displayName)
		} else {
			items = append(items, "  "+displayName)
		}
		visibleIndex++
	}
	
	content := strings.Join(items, "\n")
	m.collectionsViewport.SetContent(content)
	
	// Ensure the selected item is visible (use visible index for scrolling)
	if selectedVisibleIndex >= 0 {
		viewportHeight := m.collectionsViewport.Height
		
		// If selected item is below the visible area, scroll down
		if selectedVisibleIndex >= m.collectionsViewport.YOffset+viewportHeight {
			m.collectionsViewport.SetYOffset(selectedVisibleIndex - viewportHeight + 1)
		}
		// If selected item is above the visible area, scroll up
		if selectedVisibleIndex < m.collectionsViewport.YOffset {
			m.collectionsViewport.SetYOffset(selectedVisibleIndex)
		}
	}
}

func getMethodColor(method string) string {
	switch method {
	case "GET":
		return "🟢 GET"
	case "POST":
		return "🟡 POST"
	case "PUT":
		return "🔵 PUT"
	case "DELETE":
		return "🔴 DELETE"
	case "PATCH":
		return "🟠 PATCH"
	default:
		return "⚪ " + method
	}
}

func (m *model) startFilter(filterType FilterType) {
	m.filterMode = true
	m.filterType = filterType
	
	// Restore previous filter input
	switch filterType {
	case JQFilter:
		m.filterInput = m.lastJQFilter
		// Generate suggestions for jq filter
		m.generateJQSuggestions()
		m.showSuggestions = true
		m.selectedSuggestion = 0
	case CollectionsFilter:
		m.filterInput = m.lastCollectionsFilter
	default:
		m.filterInput = ""
	}
	
	// Set cursor to end of input
	m.filterCursorPos = len(m.filterInput)
	
	// Store original data based on filter type
	if filterType == CollectionsFilter && len(m.originalCollections) == 0 {
		m.originalCollections = make([]panels.CollectionItem, len(m.collections))
		copy(m.originalCollections, m.collections)
	}
}

func (m *model) generateJQSuggestions() {
	m.jqSuggestions = []string{}
	
	// Basic jq operations
	basicSuggestions := []string{
		".",
		".[]",
		".[0]",
		"length",
		"keys",
		"keys[]",
		"type",
		"empty",
		"map(.)",
		"select(.)",
		"sort",
		"reverse",
		"unique",
		"group_by(.)",
		"min",
		"max",
		"add",
	}
	
	m.jqSuggestions = append(m.jqSuggestions, basicSuggestions...)
	
	// Extract field suggestions from current JSON
	if m.originalResponse != "" {
		fieldSuggestions := m.extractJSONFields(m.originalResponse, "")
		m.jqSuggestions = append(m.jqSuggestions, fieldSuggestions...)
	}
}

func (m *model) extractJSONFields(jsonStr string, prefix string) []string {
	var suggestions []string
	var data interface{}
	
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return suggestions
	}
	
	suggestions = m.extractFieldsFromValue(data, prefix, 0)
	return suggestions
}

func (m *model) extractFieldsFromValue(value interface{}, prefix string, depth int) []string {
	var suggestions []string
	
	// Limit depth to avoid infinite recursion and too many suggestions
	if depth > 3 {
		return suggestions
	}
	
	switch v := value.(type) {
	case map[string]interface{}:
		for key, subValue := range v {
			fieldPath := prefix + "." + key
			if prefix == "" {
				fieldPath = "." + key
			}
			suggestions = append(suggestions, fieldPath)
			
			// Add array access for arrays
			if _, isArray := subValue.([]interface{}); isArray {
				suggestions = append(suggestions, fieldPath+"[]")
				suggestions = append(suggestions, fieldPath+"[0]")
			}
			
			// Recursively get nested fields (limited depth)
			if depth < 2 {
				nestedSuggestions := m.extractFieldsFromValue(subValue, fieldPath, depth+1)
				suggestions = append(suggestions, nestedSuggestions...)
			}
		}
	case []interface{}:
		if len(v) > 0 {
			// Get fields from first array element
			if depth < 2 {
				nestedSuggestions := m.extractFieldsFromValue(v[0], prefix+"[]", depth+1)
				suggestions = append(suggestions, nestedSuggestions...)
			}
		}
	}
	
	return suggestions
}

func (m *model) getFilteredSuggestions() []string {
	// Regenerate suggestions based on current input context
	m.generateContextualSuggestions()
	
	if m.filterInput == "" {
		return m.jqSuggestions
	}
	
	var filtered []string
	input := strings.ToLower(m.filterInput)
	
	for _, suggestion := range m.jqSuggestions {
		if strings.Contains(strings.ToLower(suggestion), input) ||
		   strings.HasPrefix(strings.ToLower(suggestion), input) {
			filtered = append(filtered, suggestion)
		}
	}
	
	return filtered
}

func (m *model) generateContextualSuggestions() {
	// Start with basic suggestions
	m.generateJQSuggestions()
	
	// Add contextual suggestions based on current input
	if m.originalResponse != "" && m.filterInput != "" {
		contextSuggestions := m.getContextualCompletions(m.filterInput)
		m.jqSuggestions = append(m.jqSuggestions, contextSuggestions...)
	}
}

func (m *model) getContextualCompletions(input string) []string {
	var suggestions []string
	
	// If input ends with a dot, suggest fields for that path
	if strings.HasSuffix(input, ".") {
		pathSuggestions := m.getFieldsForPath(input[:len(input)-1])
		for _, field := range pathSuggestions {
			suggestions = append(suggestions, input+field)
		}
	}
	
	// If input looks like a partial field path, suggest completions
	if strings.Contains(input, ".") && !strings.HasSuffix(input, ".") {
		lastDotIndex := strings.LastIndex(input, ".")
		if lastDotIndex >= 0 {
			basePath := input[:lastDotIndex]
			partialField := input[lastDotIndex+1:]
			
			pathFields := m.getFieldsForPath(basePath)
			for _, field := range pathFields {
				if strings.HasPrefix(strings.ToLower(field), strings.ToLower(partialField)) {
					suggestions = append(suggestions, basePath+"."+field)
				}
			}
		}
	}
	
	return suggestions
}

func (m *model) getFieldsForPath(path string) []string {
	var fields []string
	var data interface{}
	
	if err := json.Unmarshal([]byte(m.originalResponse), &data); err != nil {
		return fields
	}
	
	// Navigate to the specified path
	current := data
	if path != "" && path != "." {
		pathParts := strings.Split(strings.TrimPrefix(path, "."), ".")
		for _, part := range pathParts {
			if part == "" {
				continue
			}
			
			// Handle array notation
			if strings.HasSuffix(part, "[]") {
				part = part[:len(part)-2]
				if obj, ok := current.(map[string]interface{}); ok {
					if arr, ok := obj[part].([]interface{}); ok && len(arr) > 0 {
						current = arr[0] // Use first element as template
					} else {
						return fields
					}
				} else {
					return fields
				}
			} else {
				if obj, ok := current.(map[string]interface{}); ok {
					if val, exists := obj[part]; exists {
						current = val
					} else {
						return fields
					}
				} else {
					return fields
				}
			}
		}
	}
	
	// Extract fields from current object
	if obj, ok := current.(map[string]interface{}); ok {
		for key, value := range obj {
			fields = append(fields, key)
			
			// Add array notation for arrays
			if _, isArray := value.([]interface{}); isArray {
				fields = append(fields, key+"[]")
				fields = append(fields, key+"[0]")
			}
		}
	}
	
	return fields
}

func (m *model) findPreviousWordBoundary(input string, pos int) int {
	if pos <= 0 {
		return 0
	}
	
	// Word boundaries for jq expressions: ., [, ]
	wordBoundaries := []rune{'.', '[', ']'}
	
	// Start from position before cursor
	for i := pos - 1; i >= 0; i-- {
		char := rune(input[i])
		for _, boundary := range wordBoundaries {
			if char == boundary {
				// Found a boundary, position cursor after it (unless at start)
				if i == 0 {
					return 0
				}
				return i + 1
			}
		}
	}
	
	// No boundary found, go to start
	return 0
}

func (m *model) findNextWordBoundary(input string, pos int) int {
	if pos >= len(input) {
		return len(input)
	}
	
	// Word boundaries for jq expressions: ., [, ]
	wordBoundaries := []rune{'.', '[', ']'}
	
	// Start from current position
	for i := pos; i < len(input); i++ {
		char := rune(input[i])
		for _, boundary := range wordBoundaries {
			if char == boundary {
				// Found a boundary, position cursor at it
				return i
			}
		}
	}
	
	// No boundary found, go to end
	return len(input)
}

func (m *model) exitFilter() {
	// Save the current filter input before exiting
	switch m.filterType {
	case JQFilter:
		m.lastJQFilter = m.filterInput
	case CollectionsFilter:
		m.lastCollectionsFilter = m.filterInput
	}
	
	m.filterMode = false
	m.filterInput = ""
	m.showSuggestions = false
	m.selectedSuggestion = 0
	m.filterCursorPos = 0
	
	// Restore original data if needed
	if m.filterType == CollectionsFilter && len(m.originalCollections) > 0 {
		m.collections = make([]panels.CollectionItem, len(m.originalCollections))
		copy(m.collections, m.originalCollections)
		m.updateCollectionsViewport()
	}
}

func (m *model) applyFilter() tea.Cmd {
	if m.filterInput == "" {
		return nil
	}

	switch m.filterType {
	case JQFilter:
		return m.applyJqFilter()
	case CollectionsFilter:
		return m.applyCollectionsFilter()
	default:
		return nil
	}
}

func (m *model) applyJqFilter() tea.Cmd {
	if m.originalResponse == "" {
		return nil
	}

	filter := m.filterInput
	originalData := m.originalResponse

	return func() tea.Msg {
		// Parse the jq query
		query, err := gojq.Parse(filter)
		if err != nil {
			return filterMsg{filterType: JQFilter, err: fmt.Errorf("jq parse error: %v", err)}
		}

		// Parse the JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(originalData), &jsonData); err != nil {
			return filterMsg{filterType: JQFilter, err: fmt.Errorf("JSON parse error: %v", err)}
		}

		// Apply the filter
		iter := query.Run(jsonData)
		var results []interface{}
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				return filterMsg{filterType: JQFilter, err: fmt.Errorf("jq filter error: %v", err)}
			}
			results = append(results, v)
		}

		// Format the result
		var resultData interface{}
		if len(results) == 0 {
			resultData = nil
		} else if len(results) == 1 {
			resultData = results[0]
		} else {
			resultData = results
		}

		// Convert back to pretty JSON
		resultBytes, err := json.MarshalIndent(resultData, "", "  ")
		if err != nil {
			return filterMsg{filterType: JQFilter, err: fmt.Errorf("JSON marshal error: %v", err)}
		}

		return filterMsg{filterType: JQFilter, result: string(resultBytes)}
	}
}

func (m *model) applyCollectionsFilter() tea.Cmd {
	if len(m.originalCollections) == 0 {
		return nil
	}

	filter := strings.ToLower(m.filterInput)

	return func() tea.Msg {
		var filteredCollections []panels.CollectionItem
		
		for _, item := range m.originalCollections {
			// Always include folders and tag groups
			if item.IsFolder || item.IsTagGroup {
				filteredCollections = append(filteredCollections, item)
				continue
			}
			
			// Filter requests by name (case-insensitive)
			if strings.Contains(strings.ToLower(item.Name), filter) {
				filteredCollections = append(filteredCollections, item)
			}
		}

		return filterMsg{filterType: CollectionsFilter, result: fmt.Sprintf("Found %d matches", len(filteredCollections))}
	}
}

func (m *model) applyCollectionsFilterResult() {
	if len(m.originalCollections) == 0 {
		return
	}

	// If filter is empty, restore all collections and collapse them
	if m.filterInput == "" {
		m.collections = make([]panels.CollectionItem, len(m.originalCollections))
		copy(m.collections, m.originalCollections)
		
		// Reset expansion state to collapsed
		for i := range m.collections {
			if m.collections[i].IsFolder || m.collections[i].IsTagGroup {
				m.collections[i].IsExpanded = false
			}
		}
		
		// Update visibility and reset selection
		m.updateVisibility()
		if m.selectedReq >= len(m.collections) {
			m.selectedReq = 0
		}
		// Find first visible item
		for i := 0; i < len(m.collections); i++ {
			if m.collections[i].IsVisible {
				m.selectedReq = i
				break
			}
		}
		return
	}

	filter := strings.ToLower(m.filterInput)
	var filteredCollections []panels.CollectionItem
	var currentFolder *panels.CollectionItem
	var currentTagGroup *panels.CollectionItem
	var hasMatchingRequests bool
	
	for _, item := range m.originalCollections {
		if item.IsFolder {
			// Store current folder, add it later if it has matching requests
			currentFolder = &item
			currentTagGroup = nil
			hasMatchingRequests = false
		} else if item.IsTagGroup {
			// Store current tag group, add it later if it has matching requests  
			currentTagGroup = &item
			hasMatchingRequests = false
		} else {
			// This is a request - check if it matches the filter
			if strings.Contains(strings.ToLower(item.Name), filter) {
				// Add the folder if we haven't added it yet and expand it
				if currentFolder != nil && !hasMatchingRequests {
					expandedFolder := *currentFolder
					expandedFolder.IsExpanded = true
					filteredCollections = append(filteredCollections, expandedFolder)
				}
				// Add the tag group if we haven't added it yet and expand it
				if currentTagGroup != nil && !hasMatchingRequests {
					expandedTagGroup := *currentTagGroup
					expandedTagGroup.IsExpanded = true
					filteredCollections = append(filteredCollections, expandedTagGroup)
				}
				// Add the matching request
				filteredCollections = append(filteredCollections, item)
				hasMatchingRequests = true
			}
		}
	}

	m.collections = filteredCollections
	
	// Update visibility to show expanded items
	m.updateVisibility()
	
	// Reset selection to first item if current selection is out of bounds
	if m.selectedReq >= len(m.collections) {
		m.selectedReq = 0
	}
	// Find first visible item
	for i := 0; i < len(m.collections); i++ {
		if m.collections[i].IsVisible {
			m.selectedReq = i
			break
		}
	}
}

func (m model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		// Handle filter mode first (highest priority after input dialog)
		if m.filterMode {
			// Handle jq filter auto-completion navigation
			if m.filterType == JQFilter && m.showSuggestions {
				switch msg.String() {
				case "esc":
					m.exitFilter()
					return m, nil
				case "left":
					if m.filterCursorPos > 0 {
						m.filterCursorPos--
					}
					return m, nil
				case "right":
					if m.filterCursorPos < len(m.filterInput) {
						m.filterCursorPos++
					}
					return m, nil
				case "ctrl+left":
					m.filterCursorPos = m.findPreviousWordBoundary(m.filterInput, m.filterCursorPos)
					return m, nil
				case "ctrl+right":
					m.filterCursorPos = m.findNextWordBoundary(m.filterInput, m.filterCursorPos)
					return m, nil
				case "home":
					m.filterCursorPos = 0
					return m, nil
				case "end":
					m.filterCursorPos = len(m.filterInput)
					return m, nil
				case "up":
					filteredSuggestions := m.getFilteredSuggestions()
					if len(filteredSuggestions) > 0 {
						m.selectedSuggestion--
						if m.selectedSuggestion < 0 {
							m.selectedSuggestion = len(filteredSuggestions) - 1
						}
					}
					return m, nil
				case "down":
					filteredSuggestions := m.getFilteredSuggestions()
					if len(filteredSuggestions) > 0 {
						m.selectedSuggestion++
						if m.selectedSuggestion >= len(filteredSuggestions) {
							m.selectedSuggestion = 0
						}
					}
					return m, nil
				case "tab", "enter":
					// Accept selected suggestion
					filteredSuggestions := m.getFilteredSuggestions()
					if len(filteredSuggestions) > 0 && m.selectedSuggestion < len(filteredSuggestions) {
						if msg.String() == "tab" {
							// Tab just fills the suggestion
							m.filterInput = filteredSuggestions[m.selectedSuggestion]
							m.filterCursorPos = len(m.filterInput) // Set cursor to end
							m.selectedSuggestion = 0 // Reset selection
							return m, nil
						} else {
							// Enter applies the suggestion
							m.filterInput = filteredSuggestions[m.selectedSuggestion]
							cmd := m.applyFilter()
							m.exitFilter()
							return m, cmd
						}
					} else if msg.String() == "enter" {
						// No suggestion selected, just apply current input
						cmd := m.applyFilter()
						m.exitFilter()
						return m, cmd
					}
					return m, nil
				case "backspace":
					if m.filterCursorPos > 0 {
						// Remove character before cursor
						m.filterInput = m.filterInput[:m.filterCursorPos-1] + m.filterInput[m.filterCursorPos:]
						m.filterCursorPos--
						m.selectedSuggestion = 0
					}
					return m, nil
				case "delete":
					if m.filterCursorPos < len(m.filterInput) {
						// Remove character at cursor
						m.filterInput = m.filterInput[:m.filterCursorPos] + m.filterInput[m.filterCursorPos+1:]
						m.selectedSuggestion = 0
					}
					return m, nil
				default:
					// Add character to filter input at cursor position
					if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
						m.filterInput = m.filterInput[:m.filterCursorPos] + msg.String() + m.filterInput[m.filterCursorPos:]
						m.filterCursorPos++
						m.selectedSuggestion = 0
					}
					return m, nil
				}
			}
			
			// Regular filter handling for collections or jq without suggestions
			switch msg.String() {
			case "esc":
				m.exitFilter()
				return m, nil
			case "left":
				if m.filterCursorPos > 0 {
					m.filterCursorPos--
				}
				return m, nil
			case "right":
				if m.filterCursorPos < len(m.filterInput) {
					m.filterCursorPos++
				}
				return m, nil
			case "ctrl+left":
				m.filterCursorPos = m.findPreviousWordBoundary(m.filterInput, m.filterCursorPos)
				return m, nil
			case "ctrl+right":
				m.filterCursorPos = m.findNextWordBoundary(m.filterInput, m.filterCursorPos)
				return m, nil
			case "home":
				m.filterCursorPos = 0
				return m, nil
			case "end":
				m.filterCursorPos = len(m.filterInput)
				return m, nil
			case "enter":
				// For jq filter, apply the filter and exit
				if m.filterType == JQFilter {
					cmd := m.applyFilter()
					m.exitFilter()
					return m, cmd
				}
				// For collections filter, apply and exit filter mode but keep filtered results
				if m.filterType == CollectionsFilter {
					// Save the filter input
					m.lastCollectionsFilter = m.filterInput
					// Filter is already applied in real-time, just exit filter mode
					m.filterMode = false
					m.filterInput = ""
					// Don't call m.exitFilter() as that would restore original collections
					return m, nil
				}
				return m, nil
			case "backspace":
				if m.filterCursorPos > 0 {
					// Remove character before cursor
					m.filterInput = m.filterInput[:m.filterCursorPos-1] + m.filterInput[m.filterCursorPos:]
					m.filterCursorPos--
					// Apply collections filter in real-time
					if m.filterType == CollectionsFilter {
						m.applyCollectionsFilterResult()
						m.updateCollectionsViewport()
					}
				}
				return m, nil
			case "delete":
				if m.filterCursorPos < len(m.filterInput) {
					// Remove character at cursor
					m.filterInput = m.filterInput[:m.filterCursorPos] + m.filterInput[m.filterCursorPos+1:]
					// Apply collections filter in real-time
					if m.filterType == CollectionsFilter {
						m.applyCollectionsFilterResult()
						m.updateCollectionsViewport()
					}
				}
				return m, nil
			default:
				// Add character to filter input at cursor position
				if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
					m.filterInput = m.filterInput[:m.filterCursorPos] + msg.String() + m.filterInput[m.filterCursorPos:]
					m.filterCursorPos++
					// Apply collections filter in real-time
					if m.filterType == CollectionsFilter {
						m.applyCollectionsFilterResult()
						m.updateCollectionsViewport()
					}
				}
				return m, nil
			}
		}

		// Handle input dialog first (highest priority)
		if m.inputDialog.IsVisible() {
			switch msg.String() {
			case "esc":
				m.inputDialog.Hide()
				return m, nil
			case "enter":
				input, action, actionData, _ := m.inputDialog.GetResult()
				m.inputDialog.Hide()
				return m, m.executeInputCommand(action, input, actionData)
			case "tab":
				m.inputDialog.SwitchField()
				return m, nil
			case "up":
				m.inputDialog.MoveMethodSelection(-1)
				return m, nil
			case "down":
				m.inputDialog.MoveMethodSelection(1)
				return m, nil
			default:
				// Handle file picker updates for OpenAPI import
				if cmd := m.inputDialog.HandleFilePickerUpdate(msg); cmd != nil {
					return m, cmd
				}
				// Let textinput components handle all other input (including copy/paste)
				m.inputDialog.UpdateTextInputs(msg)
				return m, nil
			}
		}

		// Handle command palette
		if _, selectedCmd, handled := m.commandPalette.HandleInput(msg); handled {
			if selectedCmd != nil {
				return m, m.executeCommand(selectedCmd.Action)
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "p":
			m.commandPalette.Show()
			return m, nil
		case "tab":
			m.activePanel = (m.activePanel + 1) % 3
		case "shift+tab":
			if m.activePanel == 0 {
				m.activePanel = 2
			} else {
				m.activePanel = m.activePanel - 1
			}
		case "up":
			if m.activePanel == collectionsPanel && m.selectedReq > 0 {
				// Find previous visible item
				for i := m.selectedReq - 1; i >= 0; i-- {
					if m.collections[i].IsVisible {
						m.selectedReq = i
						break
					}
				}
				m.updateCurrentRequest()
				m.updateCollectionsViewport()
			} else if m.activePanel == requestPanel {
				if m.requestCursor > 0 {
					m.requestCursor--
				}
			} else if m.activePanel == responsePanel {
				if m.responseCursor == panels.ResponseHeadersSection {
					m.headersViewport.LineUp(1)
				} else {
					m.responseViewport.LineUp(1)
				}
			}
		case "down":
			if m.activePanel == collectionsPanel && m.selectedReq < len(m.collections)-1 {
				// Find next visible item
				for i := m.selectedReq + 1; i < len(m.collections); i++ {
					if m.collections[i].IsVisible {
						m.selectedReq = i
						break
					}
				}
				m.updateCurrentRequest()
				m.updateCollectionsViewport()
			} else if m.activePanel == requestPanel {
				maxSection := panels.GetMaxRequestSection(m.currentReq)
				if m.requestCursor < maxSection {
					m.requestCursor++
				}
			} else if m.activePanel == responsePanel {
				if m.responseCursor == panels.ResponseHeadersSection {
					m.headersViewport.LineDown(1)
				} else {
					m.responseViewport.LineDown(1)
				}
			}
		case "enter":
			if m.activePanel == collectionsPanel {
				// Check if selected item is a folder or tag group
				if m.selectedReq >= 0 && m.selectedReq < len(m.collections) {
					item := m.collections[m.selectedReq]
					if item.IsFolder || item.IsTagGroup {
						// Toggle expand/collapse
						m.toggleExpansion(m.selectedReq)
						m.updateCollectionsViewport()
					} else {
						// Regular request - update current request
						m.updateCurrentRequest()
						m.updateCollectionsViewport()
					}
				}
			} else if m.activePanel == requestPanel {
				return m, m.executeRequest()
			}
		case " ":
			if m.activePanel == requestPanel && !m.isLoading {
				return m, m.executeRequest()
			} else if m.activePanel == responsePanel {
				if m.responseCursor == panels.ResponseHeadersSection {
					m.headersViewport.HalfViewDown()
				} else {
					m.responseViewport.HalfViewDown()
				}
			}
		case "pgdown":
			if m.activePanel == collectionsPanel {
				m.collectionsViewport.HalfViewDown()
			} else if m.activePanel == responsePanel {
				if m.responseCursor == panels.ResponseHeadersSection {
					m.headersViewport.HalfViewDown()
				} else {
					m.responseViewport.HalfViewDown()
				}
			}
		case "pgup":
			if m.activePanel == collectionsPanel {
				m.collectionsViewport.HalfViewUp()
			} else if m.activePanel == responsePanel {
				if m.responseCursor == panels.ResponseHeadersSection {
					m.headersViewport.HalfViewUp()
				} else {
					m.responseViewport.HalfViewUp()
				}
			}
		case "home":
			if m.activePanel == collectionsPanel {
				m.collectionsViewport.GotoTop()
			} else if m.activePanel == responsePanel {
				if m.responseCursor == panels.ResponseHeadersSection {
					m.headersViewport.GotoTop()
				} else {
					m.responseViewport.GotoTop()
				}
			}
		case "end":
			if m.activePanel == collectionsPanel {
				m.collectionsViewport.GotoBottom()
			} else if m.activePanel == responsePanel {
				if m.responseCursor == panels.ResponseHeadersSection {
					m.headersViewport.GotoBottom()
				} else {
					m.responseViewport.GotoBottom()
				}
			}
		case "left":
			if m.activePanel == responsePanel && m.responseCursor == panels.ResponseBodySection {
				m.responseCursor = panels.ResponseHeadersSection
			}
		case "right":
			if m.activePanel == responsePanel && m.responseCursor == panels.ResponseHeadersSection {
				m.responseCursor = panels.ResponseBodySection
			}
		case "s":
			if !m.isLoading {
				return m, m.executeRequest()
			}
		case "ctrl+e":
			// Direct shortcut to edit current request
			if m.currentReq != nil {
				return m, m.executeCommand("edit_request")
			}
		case "ctrl+n":
			// Direct shortcut to create new request
			return m, m.executeCommand("new_request")
		case "ctrl+j":
			// Start jq filter for JSON responses
			if m.activePanel == responsePanel && m.lastResponse != nil && m.lastResponse.IsJSON {
				m.startFilter(JQFilter)
				return m, nil
			}
		case "ctrl+f":
			// Start collections filter
			if m.activePanel == collectionsPanel {
				m.startFilter(CollectionsFilter)
				return m, nil
			}
		case "ctrl+r":
			// Reset collections filter
			if m.activePanel == collectionsPanel && len(m.originalCollections) > 0 {
				m.collections = make([]panels.CollectionItem, len(m.originalCollections))
				copy(m.collections, m.originalCollections)
				m.originalCollections = nil
				m.selectedReq = 0
				m.lastCollectionsFilter = "" // Clear stored filter
				m.updateCollectionsViewport()
				return m, nil
			}
			// Reset jq filter
			if m.activePanel == responsePanel && m.appliedJQFilter != "" {
				// Restore original response
				m.response = m.httpClient.FormatResponseForDisplay(m.lastResponse)
				m.responseViewport.SetContent(m.response)
				m.responseViewport.GotoTop()
				m.appliedJQFilter = ""
				m.lastJQFilter = "" // Clear stored filter
				return m, nil
			}
		case "1":
			// Jump to response body
			m.activePanel = responsePanel
			m.responseCursor = panels.ResponseBodySection
			return m, nil
		}
		
	case httpResponseMsg:
		m.isLoading = false
		if msg.err != nil {
			m.response = fmt.Sprintf("Error: %v", msg.err)
			m.statusCode = 0
			m.responseViewport.SetContent(m.response)
			m.headersViewport.SetContent("No headers available")
		} else {
			m.lastResponse = msg.response
			m.originalResponse = msg.response.Body
			m.response = m.httpClient.FormatResponseForDisplay(msg.response)
			m.statusCode = msg.response.StatusCode
			m.responseViewport.SetContent(m.response)
			m.responseViewport.GotoTop()
			m.appliedJQFilter = "" // Clear applied jq filter for new response

			// Format headers for display
			var headersContent strings.Builder
			if len(msg.response.Headers) > 0 {
				// Sort headers alphabetically
				var keys []string
				for key := range msg.response.Headers {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				
				for _, key := range keys {
					headersContent.WriteString(fmt.Sprintf("%s: %s\n", key, msg.response.Headers[key]))
				}
			} else {
				headersContent.WriteString("No headers received")
			}
			m.headersViewport.SetContent(headersContent.String())
			m.headersViewport.GotoTop()
		}
		return m, nil
	case importCompleteMsg:
		if msg.success {
			m.loadBruFiles() // Refresh the collections list
		}
		// Note: Error handling could be improved with a status message
		return m, nil
	case jqFilterMsg:
		if msg.err != nil {
			// Show error in response viewport
			m.responseViewport.SetContent(fmt.Sprintf("jq Error: %v", msg.err))
		} else {
			// Show filtered result
			m.response = msg.result
			m.responseViewport.SetContent(m.response)
			m.responseViewport.GotoTop()
		}
		return m, nil
	case filterMsg:
		if msg.err != nil {
			if msg.filterType == JQFilter {
				// Show error in response viewport
				m.responseViewport.SetContent(fmt.Sprintf("jq Error: %v", msg.err))
			}
			// For collections filter errors, we could show in footer or ignore
		} else {
			if msg.filterType == JQFilter {
				// Show filtered result and save applied filter
				m.response = msg.result
				m.responseViewport.SetContent(m.response)
				m.responseViewport.GotoTop()
				m.appliedJQFilter = m.lastJQFilter // Use the saved filter from exitFilter
			} else if msg.filterType == CollectionsFilter {
				// Apply filtered collections
				m.applyCollectionsFilterResult()
				m.updateCollectionsViewport()
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *model) updateCurrentRequest() {
	if m.selectedReq <= 0 || len(m.collections) == 0 {
		return
	}

	item := m.collections[m.selectedReq]
	if item.IsFolder || item.IsTagGroup {
		return
	}

	// Use the stored RequestIndex instead of calculating it
	if item.RequestIndex >= 0 && item.RequestIndex < len(m.bruRequests) {
		m.currentReq = m.bruRequests[item.RequestIndex]
		m.requestCursor = panels.URLSection
	}
}

func (m *model) executeRequest() tea.Cmd {
	if m.currentReq == nil {
		return nil
	}

	m.isLoading = true

	return func() tea.Msg {
		response, err := m.httpClient.ExecuteRequest(m.currentReq)
		return httpResponseMsg{response: response, err: err}
	}
}

func getCurrentRequestFilePath(m *model) string {
	if m.selectedReq < 0 || m.selectedReq >= len(m.collections) {
		return ""
	}
	
	item := m.collections[m.selectedReq]
	if item.IsFolder || item.IsTagGroup {
		return ""
	}
	
	return item.FilePath
}

func getCurrentCollectionPath(m *model) string {
	collectionsDir, err := getCollectionsDir()
	if err != nil {
		return ""
	}
	
	// If no selection or invalid selection, use root collections directory
	if m.selectedReq < 0 || m.selectedReq >= len(m.collections) {
		return collectionsDir
	}
	
	currentItem := m.collections[m.selectedReq]
	
	// If current item is a folder, use that folder
	if currentItem.IsFolder {
		return currentItem.FilePath
	}
	
	// If current item is a tag group, find its parent collection folder
	if currentItem.IsTagGroup {
		for i := m.selectedReq - 1; i >= 0; i-- {
			if m.collections[i].IsFolder {
				return m.collections[i].FilePath
			}
		}
		return collectionsDir
	}
	
	// If current item is a request, find its parent collection folder
	// Look backwards to find the folder this request belongs to
	for i := m.selectedReq - 1; i >= 0; i-- {
		if m.collections[i].IsFolder {
			return m.collections[i].FilePath
		}
	}
	
	// If no parent folder found, this is a standalone request, use root
	return collectionsDir
}

func (m *model) executeCommand(action string) tea.Cmd {
	switch action {
	case "create_collection":
		spec := InputSpec{
			Type:        TextInput,
			Title:       "Create Collection",
			Prompt:      "Enter collection name:",
			Placeholder: "my-collection",
			Action:      action,
		}
		m.inputDialog.Show(spec)
		return nil
	case "new_request":
		spec := InputSpec{
			Type:   MethodURLInput,
			Title:  "Create New Request",
			Action: action,
		}
		m.inputDialog.Show(spec)
		return nil
	case "edit_request":
		if m.currentReq != nil {
			spec := InputSpec{
				Type:   MethodURLInput,
				Title:  "Edit Request",
				Action: action,
				IsEdit: true,
				PreFill: map[string]interface{}{
					"method": m.currentReq.HTTP.Method,
					"name":   m.currentReq.Meta.Name,
					"url":    m.currentReq.HTTP.URL,
				},
				ActionData: map[string]interface{}{
					"filePath": getCurrentRequestFilePath(m),
				},
			}
			m.inputDialog.Show(spec)
		}
		return nil
	case "import_openapi":
		spec := InputSpec{
			Type:   OpenAPIImportInput,
			Title:  "Import OpenAPI Specification",
			Action: action,
		}
		m.inputDialog.Show(spec)
		return nil
	default:
		return nil
	}
}

func generateRequestFilename(method, urlStr string) string {
	// Extract a meaningful name from the URL
	u, err := url.Parse(urlStr)
	if err != nil {
		// If URL parsing fails, use a simple fallback
		return fmt.Sprintf("%s-request.bru", strings.ToLower(method))
	}
	
	// Use the last part of the path or domain if path is empty
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	var name string
	if len(pathParts) > 0 && pathParts[0] != "" {
		name = pathParts[len(pathParts)-1]
	} else {
		// Use domain if no path
		name = strings.ReplaceAll(u.Host, ".", "-")
	}
	
	// Clean the name and add method prefix
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ToLower(name)
	
	return fmt.Sprintf("%s-%s.bru", strings.ToLower(method), name)
}

func (m *model) executeInputCommand(action string, input string, actionData map[string]interface{}) tea.Cmd {
	switch action {
	case "create_collection":
		if input != "" {
			collectionsDir, err := getCollectionsDir()
			if err == nil {
				collectionPath := filepath.Join(collectionsDir, input)
				err := os.MkdirAll(collectionPath, 0755)
				if err == nil {
					m.loadBruFiles() // Refresh the collections list
				}
			}
		}
		return nil
	case "new_request":
		if actionData != nil {
			method, methodOk := actionData["method"].(string)
			urlStr, urlOk := actionData["url"].(string)
			name, _ := actionData["name"].(string)
			
			if methodOk && urlOk && urlStr != "" {
				// Get the current collection path (folder or root)
				targetDir := getCurrentCollectionPath(m)
				if targetDir != "" {
					// Use display name if provided, otherwise generate from method and URL
					displayName := name
					if displayName == "" {
						displayName = fmt.Sprintf("%s %s", method, urlStr)
					}
					
					// Generate a simple filename from the URL or name
					filename := generateRequestFilename(method, urlStr)
					filePath := filepath.Join(targetDir, filename)
					
					// Create the .bru file content
					bruContent := fmt.Sprintf(`meta {
  name: %s
  type: http
  seq: 1
}

%s {
  url: %s
}
`, displayName, strings.ToLower(method), urlStr)
					
					err := os.WriteFile(filePath, []byte(bruContent), 0644)
					if err == nil {
						m.loadBruFiles() // Refresh the collections list
					}
				}
			}
		}
		return nil
	case "edit_request":
		if actionData != nil {
			method, methodOk := actionData["method"].(string)
			urlStr, urlOk := actionData["url"].(string)
			name, _ := actionData["name"].(string)
			filePath, filePathOk := actionData["filePath"].(string)
			
			if methodOk && urlOk && filePathOk && urlStr != "" && filePath != "" {
				// Use display name if provided, otherwise generate from method and URL
				displayName := name
				if displayName == "" {
					displayName = fmt.Sprintf("%s %s", method, urlStr)
				}
				
				// Create the updated .bru file content
				bruContent := fmt.Sprintf(`meta {
  name: %s
  type: http
  seq: 1
}

%s {
  url: %s
}
`, displayName, strings.ToLower(method), urlStr)
				
				err := os.WriteFile(filePath, []byte(bruContent), 0644)
				if err == nil {
					m.loadBruFiles() // Refresh the collections list
				}
			}
		}
		return nil
	case "import_openapi":
		if actionData != nil {
			source, sourceOk := actionData["source"].(string)
			collection, _ := actionData["collection"].(string)
			
			if sourceOk && source != "" {
				// Import OpenAPI spec in background and return command
				return func() tea.Msg {
					var err error
					if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
						// Import from URL
						err = ImportOpenAPIFromURL(source, collection)
					} else {
						// Import from file
						err = ImportOpenAPIFromFile(source, collection)
					}
					
					return importCompleteMsg{success: err == nil, err: err}
				}
			}
		}
		return nil
	default:
		return nil
	}
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	sidebarWidth := m.width / 3
	mainWidth := m.width - sidebarWidth - 2

	// Account for header (1 line) + footer (1 line) + some padding
	availableHeight := m.height - 4

	sidebar := panels.RenderCollections(sidebarWidth, availableHeight, m.activePanel == collectionsPanel, &m.collectionsViewport, focusedStyle, blurredStyle, titleStyle)
	main := m.renderMainArea(mainWidth, availableHeight)

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		main,
	)

	header := titleStyle.Width(m.width - 2).Render("Kalo - Bruno API Client")

	var footerText string
	if m.filterMode {
		filterName := ""
		switch m.filterType {
		case JQFilter:
			filterName = "jq"
		case CollectionsFilter:
			filterName = "search"
		}
		
		// Show suggestions for jq filter
		if m.filterType == JQFilter && m.showSuggestions {
			filteredSuggestions := m.getFilteredSuggestions()
			if len(filteredSuggestions) > 0 {
				suggestionText := ""
				// Show up to 3 suggestions in footer
				maxShow := 3
				if len(filteredSuggestions) < maxShow {
					maxShow = len(filteredSuggestions)
				}
				
				for i := 0; i < maxShow; i++ {
					if i == m.selectedSuggestion {
						suggestionText += "[" + filteredSuggestions[i] + "]"
					} else {
						suggestionText += filteredSuggestions[i]
					}
					if i < maxShow-1 {
						suggestionText += " "
					}
				}
				
				if len(filteredSuggestions) > maxShow {
					suggestionText += fmt.Sprintf(" (+%d more)", len(filteredSuggestions)-maxShow)
				}
				
				// Show input with cursor
				inputWithCursor := m.filterInput[:m.filterCursorPos] + "|" + m.filterInput[m.filterCursorPos:]
				footerText = fmt.Sprintf("%s filter: %s • ↑↓: Navigate • Tab: Complete • Enter: Apply • Esc: Cancel | %s", filterName, inputWithCursor, suggestionText)
			} else {
				// Show input with cursor
				inputWithCursor := m.filterInput[:m.filterCursorPos] + "|" + m.filterInput[m.filterCursorPos:]
				footerText = fmt.Sprintf("%s filter: %s • Enter: Apply • Esc: Cancel", filterName, inputWithCursor)
			}
		} else {
			// Show input with cursor
			inputWithCursor := m.filterInput[:m.filterCursorPos] + "|" + m.filterInput[m.filterCursorPos:]
			footerText = fmt.Sprintf("%s filter: %s • Enter: Apply • Esc: Cancel", filterName, inputWithCursor)
		}
	} else if m.isLoading {
		footerText = "Loading... • Tab: Switch panels • p: Command Palette • q: Quit"
	} else if m.activePanel == responsePanel {
		if m.appliedJQFilter != "" {
			footerText = "Tab: Switch panels • ←→: Switch Headers/Body • ↑↓: Scroll • Space/PgDn/PgUp/Home/End • s: Send • Ctrl+j: jq filter • Ctrl+r: Reset filter • p: Command Palette • q: Quit"
		} else {
			footerText = "Tab: Switch panels • ←→: Switch Headers/Body • ↑↓: Scroll • Space/PgDn/PgUp/Home/End • s: Send • Ctrl+j: jq filter • p: Command Palette • q: Quit"
		}
	} else if m.activePanel == collectionsPanel {
		if len(m.originalCollections) > 0 && len(m.collections) != len(m.originalCollections) {
			footerText = "Tab: Switch panels • ↑↓: Navigate • PgUp/PgDn/Home/End: Scroll • Enter/Space/s: Execute • Ctrl+f: Search • Ctrl+r: Reset filter • p: Command Palette • q: Quit"
		} else {
			footerText = "Tab: Switch panels • ↑↓: Navigate • PgUp/PgDn/Home/End: Scroll • Enter/Space/s: Execute • Ctrl+f: Search • p: Command Palette • q: Quit"
		}
	} else {
		footerText = "Tab: Switch panels • ↑↓: Navigate • Enter/Space/s: Execute • p: Command Palette • q: Quit"
	}
	footer := headerStyle.Width(m.width - 2).Render(footerText)

	baseView := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)

	// Overlay input dialog if visible (highest priority)
	if m.inputDialog.IsVisible() {
		inputDialogView := m.inputDialog.Render(m.width, m.height)
		return inputDialogView
	}

	// Overlay command palette if visible
	if m.commandPalette.IsVisible() {
		commandPaletteView := m.commandPalette.Render(m.width, m.height)
		return commandPaletteView
	}

	return baseView
}

func (m model) renderMainArea(width, height int) string {
	// Account for borders and titles in each panel (roughly 3 lines each)
	requestHeight := (height / 2) - 2
	responseHeight := height - requestHeight - 4

	// Ensure minimum heights
	if requestHeight < 5 {
		requestHeight = 5
	}
	if responseHeight < 5 {
		responseHeight = 5
	}

	request := panels.RenderRequest(width, requestHeight, m.currentReq, m.activePanel == requestPanel, m.requestCursor, focusedStyle, blurredStyle, titleStyle, cursorStyle, methodStyle, urlStyle, sectionStyle)
	response := panels.RenderResponse(width, responseHeight, m.activePanel == responsePanel, m.isLoading, m.lastResponse, m.statusCode, m.responseCursor, &m.headersViewport, &m.responseViewport, focusedStyle, blurredStyle, titleStyle, cursorStyle, sectionStyle, statusOkStyle, m.appliedJQFilter)

	return lipgloss.JoinVertical(lipgloss.Left, request, response)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
