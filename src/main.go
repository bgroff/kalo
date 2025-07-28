package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	filterManager    *FilterManager
	inputHandler     *InputHandler
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
		filterManager:       NewFilterManager(),
		inputHandler:        NewInputHandler(),
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

			// Sort requests by HTTP method priority (GET, POST, PUT, PATCH, DELETE, others)
			sort.Slice(requests, func(i, j int) bool {
				priorityI := getMethodPriority(requests[i].HTTP.Method)
				priorityJ := getMethodPriority(requests[j].HTTP.Method)
				if priorityI != priorityJ {
					return priorityI < priorityJ
				}
				// If same method priority, sort alphabetically by name
				return requests[i].Meta.Name < requests[j].Meta.Name
			})

			// Add requests under this tag
			for _, request := range requests {
				methodColor := getMethodColor(request.HTTP.Method)
				indentLevel := "        "
				if len(tagNames) == 1 && tagName == "untagged" {
					indentLevel = "    " // Less indentation if no tag groups
				}
				// Calculate available width for collections panel (width/3 - 4 for padding)
				availableWidth := m.width/3 - 4
				displayName := formatRequestDisplayName(indentLevel, methodColor, request.Meta.Name, request.HTTP.Method, availableWidth)

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

			// Sort requests by HTTP method priority (GET, POST, PUT, PATCH, DELETE, others)
			sort.Slice(requests, func(i, j int) bool {
				priorityI := getMethodPriority(requests[i].HTTP.Method)
				priorityJ := getMethodPriority(requests[j].HTTP.Method)
				if priorityI != priorityJ {
					return priorityI < priorityJ
				}
				// If same method priority, sort alphabetically by name
				return requests[i].Meta.Name < requests[j].Meta.Name
			})

			// Add requests under this tag
			for _, request := range requests {
				methodColor := getMethodColor(request.HTTP.Method)
				indentLevel := "    "
				if len(tagNames) == 1 && tagName == "untagged" {
					indentLevel = "  " // Less indentation if no tag groups
				}
				// Calculate available width for collections panel (width/3 - 4 for padding)
				availableWidth := m.width/3 - 4
				displayName := formatRequestDisplayName(indentLevel, methodColor, request.Meta.Name, request.HTTP.Method, availableWidth)

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
		return "🟢"
	case "POST":
		return "🟡"
	case "PUT":
		return "🔵"
	case "DELETE":
		return "🔴"
	case "PATCH":
		return "🟠"
	default:
		return "⚪"
	}
}

func getMethodPriority(method string) int {
	switch method {
	case "GET":
		return 1
	case "POST":
		return 2
	case "PUT":
		return 3
	case "PATCH":
		return 4
	case "DELETE":
		return 5
	default:
		return 6 // Other methods come last
	}
}

func formatRequestDisplayName(indentLevel, methodEmoji, requestName, method string, availableWidth int) string {
	// Create the left part: indentation + emoji + space + request name
	leftPart := indentLevel + methodEmoji + " " + requestName
	
	// Create the right part: [METHOD]
	rightPart := "[" + method + "]"
	
	// Calculate padding needed for right alignment
	usedSpace := len(leftPart) + len(rightPart)
	if usedSpace < availableWidth {
		padding := availableWidth - usedSpace
		return leftPart + strings.Repeat(" ", padding) + rightPart
	}
	
	// If not enough space, just add one space
	return leftPart + " " + rightPart
}

// Wrapper methods for filter operations
func (m *model) startFilter(filterType FilterType) {
	m.filterManager.StartFilter(filterType)
	
	// Generate suggestions for jq filter
	if filterType == JQFilter {
		m.filterManager.GenerateJQSuggestions(m.originalResponse)
	}
}

func (m *model) exitFilter() {
	restore := m.filterManager.ExitFilter()
	if len(restore) > 0 {
		m.collections = restore
		m.updateVisibility()
		m.updateCollectionsViewport()
	}
}

func (m *model) applyFilter() tea.Cmd {
	if m.filterInput() == "" {
		return nil
	}

	switch m.filterType() {
	case JQFilter:
		return m.filterManager.ApplyJQFilter(m.originalResponse)
	case CollectionsFilter:
		return nil // Collections filter is applied in real-time
	default:
		return nil
	}
}

func (m *model) applyCollectionsFilterResult() {
	filtered := m.filterManager.ApplyCollectionsFilter(m.collections)
	m.collections = filtered
	m.collections = m.filterManager.UpdateVisibility(m.collections)
	
	// Reset selection to first visible item if current selection is out of bounds
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

func (m *model) getFilteredSuggestions() []string {
	return m.filterManager.GetFilteredSuggestions(m.originalResponse)
}

func (m *model) findPreviousWordBoundary(input string, pos int) int {
	return m.filterManager.FindPreviousWordBoundary(input, pos)
}

func (m *model) findNextWordBoundary(input string, pos int) int {
	return m.filterManager.FindNextWordBoundary(input, pos)
}

// Properties for accessing filter state
func (m *model) filterMode() bool {
	return m.filterManager.mode
}

func (m *model) filterType() FilterType {
	return m.filterManager.filterType
}

func (m *model) filterInput() string {
	return m.filterManager.input
}

func (m *model) filterCursorPos() int {
	return m.filterManager.cursorPos
}

func (m *model) showSuggestions() bool {
	return m.filterManager.showSuggestions
}

func (m *model) selectedSuggestion() int {
	return m.filterManager.selectedSuggestion
}

func (m *model) appliedJQFilter() string {
	return m.filterManager.appliedJQFilter
}

func (m *model) originalCollections() []panels.CollectionItem {
	return m.filterManager.originalCollections
}

// Property setters for input handler
func (m *model) setFilterInput(input string) {
	m.filterManager.input = input
}

func (m *model) setFilterCursorPos(pos int) {
	m.filterManager.cursorPos = pos
}

func (m *model) setSelectedSuggestion(index int) {
	m.filterManager.selectedSuggestion = index
}

func (m *model) setAppliedJQFilter(filter string) {
	m.filterManager.appliedJQFilter = filter
}

func (m *model) setFilterMode(mode bool) {
	m.filterManager.mode = mode
}

func (m *model) setLastCollectionsFilter(filter string) {
	m.filterManager.lastCollectionsFilter = filter
}

func (m *model) setOriginalCollections(collections []panels.CollectionItem) {
	m.filterManager.originalCollections = collections
}

func (m *model) updateVisibility() {
	m.collections = m.filterManager.UpdateVisibility(m.collections)
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
		return m.inputHandler.HandleKeyboardInput(&m, msg)
		
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
			m.setAppliedJQFilter("") // Clear applied jq filter for new response

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
				m.setAppliedJQFilter(m.filterManager.lastJQFilter) // Use the saved filter from exitFilter
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
	if m.filterMode() {
		filterName := ""
		switch m.filterType() {
		case JQFilter:
			filterName = "jq"
		case CollectionsFilter:
			filterName = "search"
		}
		
		// Show suggestions for jq filter
		if m.filterType() == JQFilter && m.showSuggestions() {
			filteredSuggestions := m.getFilteredSuggestions()
			if len(filteredSuggestions) > 0 {
				suggestionText := ""
				// Show up to 3 suggestions in footer
				maxShow := 3
				if len(filteredSuggestions) < maxShow {
					maxShow = len(filteredSuggestions)
				}
				
				for i := 0; i < maxShow; i++ {
					if i == m.selectedSuggestion() {
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
				inputWithCursor := m.filterInput()[:m.filterCursorPos()] + "|" + m.filterInput()[m.filterCursorPos():]
				footerText = fmt.Sprintf("%s filter: %s • ↑↓: Navigate • Tab: Complete • Enter: Apply • Esc: Cancel | %s", filterName, inputWithCursor, suggestionText)
			} else {
				// Show input with cursor
				inputWithCursor := m.filterInput()[:m.filterCursorPos()] + "|" + m.filterInput()[m.filterCursorPos():]
				footerText = fmt.Sprintf("%s filter: %s • Enter: Apply • Esc: Cancel", filterName, inputWithCursor)
			}
		} else {
			// Show input with cursor
			inputWithCursor := m.filterInput()[:m.filterCursorPos()] + "|" + m.filterInput()[m.filterCursorPos():]
			footerText = fmt.Sprintf("%s filter: %s • Enter: Apply • Esc: Cancel", filterName, inputWithCursor)
		}
	} else if m.isLoading {
		footerText = "Loading... • Tab: Switch panels • p: Command Palette • q: Quit"
	} else if m.activePanel == responsePanel {
		if m.appliedJQFilter() != "" {
			footerText = "Tab: Switch panels • ←→: Switch Headers/Body • ↑↓: Scroll • Space/PgDn/PgUp/Home/End • s: Send • Ctrl+j: jq filter • Ctrl+r: Reset filter • p: Command Palette • q: Quit"
		} else {
			footerText = "Tab: Switch panels • ←→: Switch Headers/Body • ↑↓: Scroll • Space/PgDn/PgUp/Home/End • s: Send • Ctrl+j: jq filter • p: Command Palette • q: Quit"
		}
	} else if m.activePanel == collectionsPanel {
		if len(m.originalCollections()) > 0 && len(m.collections) != len(m.originalCollections()) {
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
	response := panels.RenderResponse(width, responseHeight, m.activePanel == responsePanel, m.isLoading, m.lastResponse, m.statusCode, m.responseCursor, &m.headersViewport, &m.responseViewport, focusedStyle, blurredStyle, titleStyle, cursorStyle, sectionStyle, statusOkStyle, m.appliedJQFilter())

	return lipgloss.JoinVertical(lipgloss.Left, request, response)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
