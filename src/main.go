package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	collections "kalo/src/panels/collections"
	request "kalo/src/panels/request"
	response "kalo/src/panels/response"
)

type panel int

const (
	collectionsPanel panel = iota
	requestPanel
	responsePanel
)

type httpResponseMsg struct {
	response *response.HTTPResponse
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

type FilterType = collections.FilterType

const (
	JQFilter          = collections.JQFilter
	CollectionsFilter = collections.CollectionsFilter
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
	collections      []collections.CollectionItem
	selectedReq      int
	bruRequests      []*request.BruRequest
	currentReq       *request.BruRequest
	requestCursor    request.RequestSection
	requestActiveTab int
	response         string
	statusCode       int
	httpClient       *HTTPClient
	lastResponse     *response.HTTPResponse
	isLoading        bool
	collectionsViewport viewport.Model
	responseViewport viewport.Model
	headersViewport  viewport.Model
	responseCursor   response.ResponseSection
	responseActiveTab int
	originalResponse string // Store original response for jq filtering
	commandPalette   *CommandPalette
	inputDialog      *InputDialog
	filterManager    *collections.FilterManager
	inputHandler     *InputHandler
}

// renderFilterCursor renders a solid colored cursor for filter input
func renderFilterCursor(input string, cursorPos int) string {
	if cursorPos < 0 || cursorPos > len(input) {
		return input
	}
	
	if cursorPos == len(input) {
		// Cursor at end - add a space with cursor background
		return input + currentTheme.TextCursorStyle.Render(" ")
	} else {
		// Cursor in middle - style the character at cursor position
		before := input[:cursorPos]
		cursorChar := string(input[cursorPos])
		if cursorChar == "" {
			cursorChar = " "
		}
		after := input[cursorPos+1:]
		return before + currentTheme.TextCursorStyle.Render(cursorChar) + after
	}
}

var (
	// Global theme instance
	currentTheme *Theme
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

func initialModel() *model {
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
		requestCursor:       request.QuerySection,
		responseCursor:      response.ResponseBodySection,
		statusCode:          200,
		httpClient:          NewHTTPClient(),
		collectionsViewport: collectionsVP,
		responseViewport:    responseVP,
		headersViewport:     headersVP,
		commandPalette:      NewCommandPalette(),
		inputDialog:         NewInputDialog(),
		filterManager:       collections.NewFilterManager(),
		inputHandler:        NewInputHandler(),
		response: `{
  "message": "Select a request to see response"
}`,
	}
	m.loadBruFiles()
	return &m
}

func (m *model) loadBruFiles() {
	collectionsDir, err := getCollectionsDir()
	if err != nil {
		m.collections = []collections.CollectionItem{}
		m.bruRequests = []*request.BruRequest{}
		return
	}

	data := LoadBruFiles(collectionsDir, m.width)
	m.collections = data.Collections
	m.bruRequests = data.BruRequests

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
	collections.ToggleExpansion(m.collections, index)
	
	// Update visibility of child items
	m.updateVisibility()
}


func (m *model) updateCollectionsViewport() {
	collections.UpdateCollectionsViewport(m.collections, m.selectedReq, &m.collectionsViewport, currentTheme.TextCursorStyle)
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
	m.collections = collections.UpdateVisibility(m.collections)
	
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
	return m.filterManager.Mode
}

func (m *model) filterType() FilterType {
	return m.filterManager.FilterType
}

func (m *model) filterInput() string {
	return m.filterManager.Input
}

func (m *model) filterCursorPos() int {
	return m.filterManager.CursorPos
}

func (m *model) showSuggestions() bool {
	return m.filterManager.ShowSuggestions
}

func (m *model) selectedSuggestion() int {
	return m.filterManager.SelectedSuggestion
}

func (m *model) appliedJQFilter() string {
	return m.filterManager.AppliedJQFilter
}

func (m *model) originalCollections() []collections.CollectionItem {
	return m.filterManager.OriginalCollections
}

// Property setters for input handler
func (m *model) setFilterInput(input string) {
	m.filterManager.Input = input
}

func (m *model) setFilterCursorPos(pos int) {
	m.filterManager.CursorPos = pos
}

func (m *model) setSelectedSuggestion(index int) {
	m.filterManager.SelectedSuggestion = index
}

func (m *model) setAppliedJQFilter(filter string) {
	m.filterManager.AppliedJQFilter = filter
}

func (m *model) setFilterMode(mode bool) {
	m.filterManager.Mode = mode
}

func (m *model) setLastCollectionsFilter(filter string) {
	m.filterManager.LastCollectionsFilter = filter
}

func (m *model) setOriginalCollections(collections []collections.CollectionItem) {
	m.filterManager.OriginalCollections = collections
}

func (m *model) updateVisibility() {
	m.collections = collections.UpdateVisibility(m.collections)
}


func (m *model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.inputHandler.HandleKeyboardInput(m, msg)
		
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
				m.setAppliedJQFilter(m.filterManager.LastJQFilter) // Use the saved filter from exitFilter
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
		m.requestCursor = request.QuerySection
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
					"tags":   m.currentReq.Tags,
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
	case "switch_theme":
		spec := InputSpec{
			Type:   ThemeSelectionInput,
			Title:  "Switch Theme",
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
			tags, _ := actionData["tags"].([]string)
			
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
					
					// Create a new BruRequest object
					newReq := &request.BruRequest{
						Meta: request.BruMeta{
							Name: displayName,
							Type: "http",
							Seq:  1,
						},
						HTTP: request.BruHTTP{
							Method: method,
							URL:    urlStr,
						},
						Tags: tags,
					}
					
					// Generate complete .bru file content
					bruContent := m.generateBruContent(newReq)
					
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
			tags, _ := actionData["tags"].([]string)
			filePath, filePathOk := actionData["filePath"].(string)
			
			if methodOk && urlOk && filePathOk && urlStr != "" && filePath != "" {
				// Use display name if provided, otherwise generate from method and URL
				displayName := name
				if displayName == "" {
					displayName = fmt.Sprintf("%s %s", method, urlStr)
				}
				
				// Update the current request with new values
				if m.currentReq != nil {
					m.currentReq.Meta.Name = displayName
					m.currentReq.HTTP.Method = method
					m.currentReq.HTTP.URL = urlStr
					if tags != nil {
						m.currentReq.Tags = tags
					}
					
					// Generate complete .bru file content preserving all existing data
					bruContent := m.generateBruContent(m.currentReq)
					
					err := os.WriteFile(filePath, []byte(bruContent), 0644)
					if err == nil {
						m.loadBruFiles() // Refresh the collections list
					}
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
	case "switch_theme":
		if actionData != nil {
			themeName, themeOk := actionData["theme"].(string)
			if themeOk && themeName != "" {
				// Switch to the selected theme
				currentTheme = ReloadTheme(themeName)
				// Note: The UI will automatically update on next render
			}
		}
		return nil
	default:
		return nil
	}
}

func (m *model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	sidebarWidth := m.width / 3
	mainWidth := m.width - sidebarWidth - 2

	// Account for header (1 line) + footer (1 line) + titles + some padding
	availableHeight := m.height - 5

	// Render collections title separately above the sidebar
	collectionsTitle := m.renderCollectionsTitle(sidebarWidth)
	sidebar := collections.RenderCollections(sidebarWidth, availableHeight, m.activePanel == collectionsPanel, &m.collectionsViewport, currentTheme.FocusedStyle, currentTheme.BlurredStyle, currentTheme.TitleStyle)
	
	// Combine collections title with sidebar
	sidebarWithTitle := lipgloss.JoinVertical(lipgloss.Left, collectionsTitle, sidebar)
	
	main := m.renderMainArea(mainWidth, availableHeight)

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarWithTitle,
		main,
	)

	header := currentTheme.TitleStyle.Width(m.width - 2).Render("Kalo - Bruno API Client")

	// Get footer text from input handler (which delegates to appropriate panel)
	footerText := m.inputHandler.GetFooterText(m)
	footer := currentTheme.HeaderStyle.Width(m.width - 2).Render(footerText)

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

func (m *model) renderMainArea(width, height int) string {
	// Account for borders and separate titles (roughly 4 lines total)
	titleHeight := 1
	requestHeight := (height - 2*titleHeight) / 2 - 2
	responseHeight := height - requestHeight - 2*titleHeight - 4

	// Ensure minimum heights
	if requestHeight < 5 {
		requestHeight = 5
	}
	if responseHeight < 5 {
		responseHeight = 5
	}

	// Render titles separately above each panel
	requestTitle := m.renderRequestTitle(width)
	responseTitle := m.renderResponseTitle(width)

	request := request.RenderRequest(width, requestHeight, m.currentReq, m.activePanel == requestPanel, m.requestCursor, m.requestActiveTab, currentTheme.FocusedStyle, currentTheme.BlurredStyle, currentTheme.TitleStyle, currentTheme.CursorStyle, currentTheme.MethodStyle, currentTheme.URLStyle, currentTheme.SectionStyle, currentTheme.TextCursorStyle)
	response := response.RenderResponse(width, responseHeight, m.activePanel == responsePanel, m.isLoading, m.lastResponse, m.statusCode, m.responseCursor, m.responseActiveTab, &m.headersViewport, &m.responseViewport, currentTheme.FocusedStyle, currentTheme.BlurredStyle, currentTheme.TitleStyle, currentTheme.CursorStyle, currentTheme.SectionStyle, currentTheme.StatusOkStyle, m.appliedJQFilter())

	return lipgloss.JoinVertical(lipgloss.Left, requestTitle, request, responseTitle, response)
}

func (m *model) renderRequestTitle(width int) string {
	var titleContent string
	
	if m.currentReq == nil {
		titleContent = " Request "
	} else {
		// Create title with method and URL
		titleContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			" Request ",
			" ",
			currentTheme.MethodStyle.Render(m.currentReq.HTTP.Method),
			" ",
			currentTheme.URLStyle.Render(m.currentReq.HTTP.URL),
			" ",
		)
	}
	
	return currentTheme.TitleStyle.Width(width-2).Render(titleContent)
}

func (m *model) renderResponseTitle(width int) string {
	var titleContent string
	
	if m.isLoading {
		titleContent = " Response ⏳ Loading..."
	} else if m.lastResponse != nil {
		// Extract MIME type from Content-Type header
		contentType := ""
		if ct, exists := m.lastResponse.Headers["Content-Type"]; exists {
			// Extract just the MIME type part (before semicolon if present)
			if idx := strings.Index(ct, ";"); idx != -1 {
				contentType = strings.TrimSpace(ct[:idx])
			} else {
				contentType = strings.TrimSpace(ct)
			}
		}
		
		var statusStyle lipgloss.Style
		if m.statusCode >= 200 && m.statusCode < 300 {
			statusStyle = currentTheme.StatusOkStyle
		} else if m.statusCode >= 400 {
			statusStyle = lipgloss.NewStyle().Background(lipgloss.Color("196")).Foreground(lipgloss.Color("0")).Padding(0, 1)
		} else {
			statusStyle = lipgloss.NewStyle().Background(lipgloss.Color("214")).Foreground(lipgloss.Color("0")).Padding(0, 1)
		}
		
		statusText := fmt.Sprintf("%d %s", m.statusCode, http.StatusText(m.statusCode))
		
		// Add timing information
		timing := ""
		if m.lastResponse.ResponseTime > 0 {
			if m.lastResponse.ResponseTime < time.Millisecond {
				timing = fmt.Sprintf(" • %.2fμs", float64(m.lastResponse.ResponseTime.Nanoseconds())/1000)
			} else if m.lastResponse.ResponseTime < time.Second {
				timing = fmt.Sprintf(" • %.2fms", float64(m.lastResponse.ResponseTime.Nanoseconds())/1000000)
			} else {
				timing = fmt.Sprintf(" • %.2fs", m.lastResponse.ResponseTime.Seconds())
			}
		}
		
		// Add MIME type info
		mimeInfo := ""
		if contentType != "" {
			mimeInfo = fmt.Sprintf(" • %s", contentType)
		}
		
		titleContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			" Response ",
			statusStyle.Render(statusText),
			timing,
			mimeInfo,
			" ",
		)
	} else {
		titleContent = " Response " + currentTheme.StatusOkStyle.Render("200 OK") + " • Mock Response"
	}
	
	return currentTheme.TitleStyle.Width(width-2).Render(titleContent)
}

func (m *model) renderCollectionsTitle(width int) string {
	return currentTheme.TitleStyle.Width(width-2).Render(" Collections ")
}

// generateBruContent creates a complete .bru file content from a BruRequest,
// preserving all existing data including tags, headers, query params, body, auth, etc.
func (m *model) generateBruContent(request *request.BruRequest) string {
	var content strings.Builder
	
	// Meta block
	content.WriteString("meta {\n")
	content.WriteString(fmt.Sprintf("  name: %s\n", request.Meta.Name))
	content.WriteString("  type: http\n")
	content.WriteString(fmt.Sprintf("  seq: %d\n", request.Meta.Seq))
	content.WriteString("}\n\n")
	
	// Tags block (if tags exist)
	if len(request.Tags) > 0 {
		content.WriteString("tags {\n")
		for _, tag := range request.Tags {
			content.WriteString(fmt.Sprintf("  %s\n", tag))
		}
		content.WriteString("}\n\n")
	}
	
	// HTTP method block
	content.WriteString(fmt.Sprintf("%s {\n", strings.ToLower(request.HTTP.Method)))
	content.WriteString(fmt.Sprintf("  url: %s\n", request.HTTP.URL))
	content.WriteString("}\n")
	
	// Query parameters
	if len(request.Query) > 0 {
		content.WriteString("\nquery {\n")
		// Sort keys for consistent output
		keys := make([]string, 0, len(request.Query))
		for key := range request.Query {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			content.WriteString(fmt.Sprintf("  %s: %s\n", key, request.Query[key]))
		}
		content.WriteString("}\n")
	}
	
	// Headers
	if len(request.Headers) > 0 {
		content.WriteString("\nheaders {\n")
		// Sort keys for consistent output
		keys := make([]string, 0, len(request.Headers))
		for key := range request.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			content.WriteString(fmt.Sprintf("  %s: %s\n", key, request.Headers[key]))
		}
		content.WriteString("}\n")
	}
	
	// Auth
	if request.Auth.Type != "" {
		content.WriteString("\nauth {\n")
		content.WriteString(fmt.Sprintf("  mode: %s\n", request.Auth.Type))
		if len(request.Auth.Values) > 0 {
			// Sort keys for consistent output
			keys := make([]string, 0, len(request.Auth.Values))
			for key := range request.Auth.Values {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				content.WriteString(fmt.Sprintf("  %s: %s\n", key, request.Auth.Values[key]))
			}
		}
		content.WriteString("}\n")
	}
	
	// Body
	if request.Body.Type != "" && request.Body.Data != "" {
		content.WriteString(fmt.Sprintf("\nbody:%s {\n", request.Body.Type))
		content.WriteString(request.Body.Data)
		content.WriteString("\n}\n")
	}
	
	// Variables
	if len(request.Vars) > 0 {
		content.WriteString("\nvars {\n")
		// Sort keys for consistent output
		keys := make([]string, 0, len(request.Vars))
		for key := range request.Vars {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			content.WriteString(fmt.Sprintf("  %s: %s\n", key, request.Vars[key]))
		}
		content.WriteString("}\n")
	}
	
	// Tests
	if request.Tests != "" {
		content.WriteString("\ntests {\n")
		content.WriteString(request.Tests)
		content.WriteString("\n}\n")
	}
	
	// Docs
	if request.Docs != "" {
		content.WriteString("\ndocs {\n")
		content.WriteString(request.Docs)
		content.WriteString("\n}\n")
	}
	
	return content.String()
}

func main() {
	// Initialize theme system
	currentTheme = LoadTheme("default") // Can be configurable later
	
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
