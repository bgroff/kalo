package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"kalo/src/panels"
)

type InputHandler struct{}

func NewInputHandler() *InputHandler {
	return &InputHandler{}
}

func (h *InputHandler) HandleKeyboardInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle filter mode first (highest priority after input dialog)
	if m.filterMode() {
		return h.handleFilterInput(m, msg)
	}

	// Handle input dialog first (highest priority)
	if m.inputDialog.IsVisible() {
		return h.handleDialogInput(m, msg)
	}

	// Handle text input modes with high priority (before command palette and normal shortcuts)
	if h.isInTextInputMode(m) {
		return h.handleTextInputMode(m, msg)
	}

	// Handle command palette
	if _, selectedCmd, handled := m.commandPalette.HandleInput(msg); handled {
		return h.handleCommandPalette(m, selectedCmd)
	}

	// Normal navigation and shortcuts
	return h.handleNormalInput(m, msg)
}

// isInTextInputMode checks if we're currently in a text input editing mode
func (h *InputHandler) isInTextInputMode(m *model) bool {
	// Check for query parameter text input mode
	if m.activePanel == requestPanel && m.requestCursor == panels.QuerySection && m.currentReq != nil {
		if m.currentReq.QueryEditState != nil {
			mode := m.currentReq.QueryEditState.Mode
			return mode == panels.QueryEditKeyMode || mode == panels.QueryEditValueMode || mode == panels.QueryAddMode
		}
	}
	
	// Check for tag text input mode
	if m.activePanel == requestPanel && m.requestCursor == panels.TagsSection && m.currentReq != nil {
		if m.currentReq.TagEditState != nil {
			mode := m.currentReq.TagEditState.Mode
			return mode == panels.TagEditingMode || mode == panels.TagAddMode
		}
	}
	
	// Check for header text input mode
	if m.activePanel == requestPanel && m.requestCursor == panels.HeadersSection && m.currentReq != nil {
		if m.currentReq.HeaderEditState != nil {
			mode := m.currentReq.HeaderEditState.Mode
			return mode == panels.HeaderEditKeyMode || mode == panels.HeaderEditValueMode || mode == panels.HeaderAddMode
		}
	}
	
	// Add other text input modes here as needed (e.g., request body editing)
	// if m.activePanel == requestPanel && m.requestCursor == panels.BodySection && m.isEditingBody {
	//     return true
	// }
	
	return false
}

// handleTextInputMode handles input when in a text input editing mode
func (h *InputHandler) handleTextInputMode(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Only allow essential navigation keys to interrupt text input mode
	switch msg.String() {
	case "esc":
		// Always allow escape to exit text input mode
		return h.handleNormalInput(m, msg)
	case "enter":
		// Allow enter for completing text input
		return h.handleNormalInput(m, msg)
	case "tab", "shift+tab":
		// Allow tab navigation within text input contexts
		return h.handleNormalInput(m, msg)
	case "ctrl+c":
		// Always allow quit
		return h.handleNormalInput(m, msg)
	case "p":
		// Allow command palette to be opened
		return h.handleNormalInput(m, msg)
	default:
		// For all other input, delegate to specific text input handlers
		if m.activePanel == requestPanel && m.requestCursor == panels.QuerySection && m.currentReq != nil {
			return h.handleQueryParameterTextInput(m, msg)
		}
		
		if m.activePanel == requestPanel && m.requestCursor == panels.TagsSection && m.currentReq != nil {
			return h.handleTagTextInput(m, msg)
		}
		
		if m.activePanel == requestPanel && m.requestCursor == panels.HeadersSection && m.currentReq != nil {
			return h.handleHeaderTextInput(m, msg)
		}
		
		// Add other text input handlers here as needed
		// if m.activePanel == requestPanel && m.requestCursor == panels.BodySection {
		//     return h.handleRequestBodyTextInput(m, msg)
		// }
		
		// Fallback to normal input handling
		return h.handleNormalInput(m, msg)
	}
}

func (h *InputHandler) handleFilterInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle jq filter auto-completion navigation
	if m.filterType() == JQFilter && m.showSuggestions() {
		switch msg.String() {
		case "esc":
			m.exitFilter()
			return *m, nil
		case "left":
			if m.filterCursorPos() > 0 {
				m.setFilterCursorPos(m.filterCursorPos() - 1)
			}
			return *m, nil
		case "right":
			if m.filterCursorPos() < len(m.filterInput()) {
				m.setFilterCursorPos(m.filterCursorPos() + 1)
			}
			return *m, nil
		case "ctrl+left":
			m.setFilterCursorPos(m.findPreviousWordBoundary(m.filterInput(), m.filterCursorPos()))
			return *m, nil
		case "ctrl+right":
			m.setFilterCursorPos(m.findNextWordBoundary(m.filterInput(), m.filterCursorPos()))
			return *m, nil
		case "home":
			m.setFilterCursorPos(0)
			return *m, nil
		case "end":
			m.setFilterCursorPos(len(m.filterInput()))
			return *m, nil
		case "up":
			filteredSuggestions := m.getFilteredSuggestions()
			if len(filteredSuggestions) > 0 {
				selected := m.selectedSuggestion() - 1
				if selected < 0 {
					selected = len(filteredSuggestions) - 1
				}
				m.setSelectedSuggestion(selected)
			}
			return *m, nil
		case "down":
			filteredSuggestions := m.getFilteredSuggestions()
			if len(filteredSuggestions) > 0 {
				selected := m.selectedSuggestion() + 1
				if selected >= len(filteredSuggestions) {
					selected = 0
				}
				m.setSelectedSuggestion(selected)
			}
			return *m, nil
		case "tab", "enter":
			// Accept selected suggestion
			filteredSuggestions := m.getFilteredSuggestions()
			if len(filteredSuggestions) > 0 && m.selectedSuggestion() < len(filteredSuggestions) {
				if msg.String() == "tab" {
					// Tab just fills the suggestion
					m.setFilterInput(filteredSuggestions[m.selectedSuggestion()])
					m.setFilterCursorPos(len(m.filterInput())) // Set cursor to end
					m.setSelectedSuggestion(0) // Reset selection
					return *m, nil
				} else {
					// Enter applies the suggestion
					m.setFilterInput(filteredSuggestions[m.selectedSuggestion()])
					cmd := m.applyFilter()
					m.exitFilter()
					return *m, cmd
				}
			} else if msg.String() == "enter" {
				// No suggestion selected, just apply current input
				cmd := m.applyFilter()
				m.exitFilter()
				return *m, cmd
			}
			return *m, nil
		case "backspace":
			if m.filterCursorPos() > 0 {
				// Remove character before cursor
				input := m.filterInput()
				newInput := input[:m.filterCursorPos()-1] + input[m.filterCursorPos():]
				m.setFilterInput(newInput)
				m.setFilterCursorPos(m.filterCursorPos() - 1)
				m.setSelectedSuggestion(0)
			}
			return *m, nil
		case "delete":
			if m.filterCursorPos() < len(m.filterInput()) {
				// Remove character at cursor
				input := m.filterInput()
				newInput := input[:m.filterCursorPos()] + input[m.filterCursorPos()+1:]
				m.setFilterInput(newInput)
				m.setSelectedSuggestion(0)
			}
			return *m, nil
		default:
			// Add character to filter input at cursor position
			if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
				input := m.filterInput()
				newInput := input[:m.filterCursorPos()] + msg.String() + input[m.filterCursorPos():]
				m.setFilterInput(newInput)
				m.setFilterCursorPos(m.filterCursorPos() + 1)
				m.setSelectedSuggestion(0)
			}
			return *m, nil
		}
	}
	
	// Regular filter handling for collections or jq without suggestions
	switch msg.String() {
	case "esc":
		m.exitFilter()
		return *m, nil
	case "left":
		if m.filterCursorPos() > 0 {
			m.setFilterCursorPos(m.filterCursorPos() - 1)
		}
		return *m, nil
	case "right":
		if m.filterCursorPos() < len(m.filterInput()) {
			m.setFilterCursorPos(m.filterCursorPos() + 1)
		}
		return *m, nil
	case "ctrl+left":
		m.setFilterCursorPos(m.findPreviousWordBoundary(m.filterInput(), m.filterCursorPos()))
		return *m, nil
	case "ctrl+right":
		m.setFilterCursorPos(m.findNextWordBoundary(m.filterInput(), m.filterCursorPos()))
		return *m, nil
	case "home":
		m.setFilterCursorPos(0)
		return *m, nil
	case "end":
		m.setFilterCursorPos(len(m.filterInput()))
		return *m, nil
	case "enter":
		// For jq filter, apply the filter and exit
		if m.filterType() == JQFilter {
			cmd := m.applyFilter()
			m.exitFilter()
			return *m, cmd
		}
		// For collections filter, apply and exit filter mode but keep filtered results
		if m.filterType() == CollectionsFilter {
			// Save the filter input
			m.setLastCollectionsFilter(m.filterInput())
			// Filter is already applied in real-time, just exit filter mode
			m.setFilterMode(false)
			m.setFilterInput("")
			// Don't call m.exitFilter() as that would restore original collections
			return *m, nil
		}
		return *m, nil
	case "backspace":
		if m.filterCursorPos() > 0 {
			// Remove character before cursor
			input := m.filterInput()
			newInput := input[:m.filterCursorPos()-1] + input[m.filterCursorPos():]
			m.setFilterInput(newInput)
			m.setFilterCursorPos(m.filterCursorPos() - 1)
			// Apply collections filter in real-time
			if m.filterType() == CollectionsFilter {
				m.applyCollectionsFilterResult()
				m.updateCollectionsViewport()
			}
		}
		return *m, nil
	case "delete":
		if m.filterCursorPos() < len(m.filterInput()) {
			// Remove character at cursor
			input := m.filterInput()
			newInput := input[:m.filterCursorPos()] + input[m.filterCursorPos()+1:]
			m.setFilterInput(newInput)
			// Apply collections filter in real-time
			if m.filterType() == CollectionsFilter {
				m.applyCollectionsFilterResult()
				m.updateCollectionsViewport()
			}
		}
		return *m, nil
	default:
		// Add character to filter input at cursor position
		if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
			input := m.filterInput()
			newInput := input[:m.filterCursorPos()] + msg.String() + input[m.filterCursorPos():]
			m.setFilterInput(newInput)
			m.setFilterCursorPos(m.filterCursorPos() + 1)
			// Apply collections filter in real-time
			if m.filterType() == CollectionsFilter {
				m.applyCollectionsFilterResult()
				m.updateCollectionsViewport()
			}
		}
		return *m, nil
	}
}

func (h *InputHandler) handleDialogInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputDialog.Hide()
		return *m, nil
	case "enter":
		m.inputDialog.Confirm()
		input, action, actionData, _ := m.inputDialog.GetResult()
		m.inputDialog.Hide()
		return *m, m.executeInputCommand(action, input, actionData)
	case "tab":
		m.inputDialog.SwitchField()
		return *m, nil
	case "up":
		m.inputDialog.MoveMethodSelection(-1)
		return *m, nil
	case "down":
		m.inputDialog.MoveMethodSelection(1)
		return *m, nil
	default:
		// Handle file picker updates for OpenAPI import
		if cmd := m.inputDialog.HandleFilePickerUpdate(msg); cmd != nil {
			return *m, cmd
		}
		// Let textinput components handle all other input (including copy/paste)
		m.inputDialog.UpdateTextInputs(msg)
		return *m, nil
	}
}

func (h *InputHandler) handleCommandPalette(m *model, selectedCmd *Command) (tea.Model, tea.Cmd) {
	if selectedCmd != nil {
		return *m, m.executeCommand(selectedCmd.Action)
	}
	return *m, nil
}

func (h *InputHandler) handleNormalInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return *m, tea.Quit
	case "p":
		m.commandPalette.Show()
		return *m, nil
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
			return h.handleRequestPanelUp(m)
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
			return h.handleRequestPanelDown(m)
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
			return h.handleRequestPanelEnter(m)
		}
	case " ":
		if m.activePanel == requestPanel && !m.isLoading {
			return *m, m.executeRequest()
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
		if m.activePanel == responsePanel {
			// Navigate response tabs left
			if m.responseActiveTab > 0 {
				m.responseActiveTab--
			} else {
				m.responseActiveTab = len(panels.GetResponseTabNames()) - 1 // Wrap to last tab
			}
		} else if m.activePanel == requestPanel {
			// Navigate request tabs left
			if m.requestActiveTab > 0 {
				m.requestActiveTab--
			} else {
				m.requestActiveTab = len(panels.GetRequestTabNames()) - 1 // Wrap to last tab
			}
			// Update request cursor to match active tab
			m.requestCursor = panels.GetRequestTabSection(m.requestActiveTab)
		}
	case "right":
		if m.activePanel == responsePanel {
			// Navigate response tabs right
			maxTabs := len(panels.GetResponseTabNames())
			if m.responseActiveTab < maxTabs-1 {
				m.responseActiveTab++
			} else {
				m.responseActiveTab = 0 // Wrap to first tab
			}
		} else if m.activePanel == requestPanel {
			// Navigate request tabs right
			maxTabs := len(panels.GetRequestTabNames())
			if m.requestActiveTab < maxTabs-1 {
				m.requestActiveTab++
			} else {
				m.requestActiveTab = 0 // Wrap to first tab
			}
			// Update request cursor to match active tab
			m.requestCursor = panels.GetRequestTabSection(m.requestActiveTab)
		}
	case "s":
		if !m.isLoading {
			return *m, m.executeRequest()
		}
	case "ctrl+e":
		// Direct shortcut to edit current request
		if m.currentReq != nil {
			return *m, m.executeCommand("edit_request")
		}
	case "ctrl+n":
		// Direct shortcut to create new request
		return *m, m.executeCommand("new_request")
	case "ctrl+j":
		// Start jq filter for JSON responses
		if m.activePanel == responsePanel && m.lastResponse != nil && m.lastResponse.IsJSON {
			m.startFilter(JQFilter)
			return *m, nil
		}
	case "ctrl+f":
		// Start collections filter
		if m.activePanel == collectionsPanel {
			m.startFilter(CollectionsFilter)
			return *m, nil
		}
	case "ctrl+r":
		// Reset collections filter
		if m.activePanel == collectionsPanel && len(m.originalCollections()) > 0 {
			m.collections = make([]panels.CollectionItem, len(m.originalCollections()))
			copy(m.collections, m.originalCollections())
			m.setOriginalCollections(nil)
			m.selectedReq = 0
			m.filterManager.Reset(CollectionsFilter) // Clear stored filter
			m.updateCollectionsViewport()
			return *m, nil
		}
		// Reset jq filter
		if m.activePanel == responsePanel && m.appliedJQFilter() != "" {
			// Restore original response
			m.response = m.httpClient.FormatResponseForDisplay(m.lastResponse)
			m.responseViewport.SetContent(m.response)
			m.responseViewport.GotoTop()
			m.setAppliedJQFilter("")
			m.filterManager.Reset(JQFilter) // Clear stored filter
			return *m, nil
		}
	case "1":
		// Jump to response body
		m.activePanel = responsePanel
		m.responseCursor = panels.ResponseBodySection
		return *m, nil
	case "a":
		// Add query parameter
		if m.activePanel == requestPanel && m.requestCursor == panels.QuerySection && m.currentReq != nil {
			return h.handleQueryParameterAdd(m)
		}
		// Add tag
		if m.activePanel == requestPanel && m.requestCursor == panels.TagsSection && m.currentReq != nil {
			return h.handleTagAdd(m)
		}
		// Add header
		if m.activePanel == requestPanel && m.requestCursor == panels.HeadersSection && m.currentReq != nil {
			return h.handleHeaderAdd(m)
		}
	case "i":
		// Edit query parameter
		if m.activePanel == requestPanel && m.requestCursor == panels.QuerySection && m.currentReq != nil {
			return h.handleQueryParameterEdit(m)
		}
		// Edit tag
		if m.activePanel == requestPanel && m.requestCursor == panels.TagsSection && m.currentReq != nil {
			return h.handleTagEdit(m)
		}
		// Edit header
		if m.activePanel == requestPanel && m.requestCursor == panels.HeadersSection && m.currentReq != nil {
			return h.handleHeaderEdit(m)
		}
	case "d":
		// Delete query parameter
		if m.activePanel == requestPanel && m.requestCursor == panels.QuerySection && m.currentReq != nil {
			return h.handleQueryParameterDelete(m)
		}
		// Delete tag
		if m.activePanel == requestPanel && m.requestCursor == panels.TagsSection && m.currentReq != nil {
			return h.handleTagDelete(m)
		}
		// Delete header
		if m.activePanel == requestPanel && m.requestCursor == panels.HeadersSection && m.currentReq != nil {
			return h.handleHeaderDelete(m)
		}
	case "y", "n":
		// Handle deletion confirmation
		if m.activePanel == requestPanel && m.requestCursor == panels.QuerySection && m.currentReq != nil {
			return h.handleQueryParameterDeleteConfirm(m, msg.String() == "y")
		}
		// Handle tag deletion confirmation
		if m.activePanel == requestPanel && m.requestCursor == panels.TagsSection && m.currentReq != nil {
			return h.handleTagDeleteConfirm(m, msg.String() == "y")
		}
		// Handle header deletion confirmation
		if m.activePanel == requestPanel && m.requestCursor == panels.HeadersSection && m.currentReq != nil {
			return h.handleHeaderDeleteConfirm(m, msg.String() == "y")
		}
	case "esc":
		// Cancel query parameter editing
		if m.activePanel == requestPanel && m.requestCursor == panels.QuerySection && m.currentReq != nil {
			return h.handleQueryParameterCancel(m)
		}
		// Cancel tag editing
		if m.activePanel == requestPanel && m.requestCursor == panels.TagsSection && m.currentReq != nil {
			return h.handleTagCancel(m)
		}
		// Cancel header editing
		if m.activePanel == requestPanel && m.requestCursor == panels.HeadersSection && m.currentReq != nil {
			return h.handleHeaderCancel(m)
		}
	}
	
	return *m, nil
}

// handleRequestPanelEnter handles enter key in request panel
func (h *InputHandler) handleRequestPanelEnter(m *model) (tea.Model, tea.Cmd) {
	if m.requestCursor == panels.QuerySection && m.currentReq != nil {
		return h.handleQueryParameterEnter(m)
	}
	if m.requestCursor == panels.TagsSection && m.currentReq != nil {
		return h.handleTagEnter(m)
	}
	if m.requestCursor == panels.HeadersSection && m.currentReq != nil {
		return h.handleHeaderEnter(m)
	}
	return *m, m.executeRequest()
}

// handleRequestPanelUp handles up key in request panel
func (h *InputHandler) handleRequestPanelUp(m *model) (tea.Model, tea.Cmd) {
	if m.requestCursor == panels.QuerySection && m.currentReq != nil {
		return h.handleQueryParameterUp(m)
	}
	if m.requestCursor == panels.TagsSection && m.currentReq != nil {
		return h.handleTagUp(m)
	}
	if m.requestCursor == panels.HeadersSection && m.currentReq != nil {
		// Check if we can handle header up navigation
		m.currentReq.InitializeHeaderEditState()
		if m.currentReq.HeaderEditState.Mode == panels.HeaderViewMode && len(m.currentReq.HeaderEditState.Parameters) > 0 && m.currentReq.HeaderEditState.SelectedIndex > 0 {
			return h.handleHeaderUp(m)
		}
	}
	if m.requestCursor > 0 {
		m.requestCursor--
	}
	return *m, nil
}

// handleRequestPanelDown handles down key in request panel
func (h *InputHandler) handleRequestPanelDown(m *model) (tea.Model, tea.Cmd) {
	if m.requestCursor == panels.QuerySection && m.currentReq != nil {
		return h.handleQueryParameterDown(m)
	}
	if m.requestCursor == panels.TagsSection && m.currentReq != nil {
		return h.handleTagDown(m)
	}
	if m.requestCursor == panels.HeadersSection && m.currentReq != nil {
		m.currentReq.InitializeHeaderEditState()
		// Only handle header navigation if there are headers, we're in view mode, and can navigate down
		if m.currentReq.HeaderEditState.Mode == panels.HeaderViewMode && len(m.currentReq.HeaderEditState.Parameters) > 0 && m.currentReq.HeaderEditState.SelectedIndex < len(m.currentReq.HeaderEditState.Parameters)-1 {
			return h.handleHeaderDown(m)
		}
		// If we're in headers section but can't navigate down within headers, allow general navigation
	}
	maxSection := panels.GetMaxRequestSection(m.currentReq)
	if m.requestCursor < maxSection {
		m.requestCursor++
	}
	return *m, nil
}

// handleQueryParameterEnter handles enter key in query parameter editing
func (h *InputHandler) handleQueryParameterEnter(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeQueryEditState()
	editState := m.currentReq.QueryEditState
	
	switch editState.Mode {
	case panels.QueryViewMode:
		// Start editing current parameter key
		if len(editState.Parameters) > 0 && editState.SelectedIndex < len(editState.Parameters) {
			param := editState.Parameters[editState.SelectedIndex]
			editState.Mode = panels.QueryEditKeyMode
			editState.EditingKey = param.Key
			editState.OriginalKey = param.Key
			editState.CursorPos = len(param.Key)
		}
	case panels.QueryEditKeyMode:
		// Move to editing value
		editState.Mode = panels.QueryEditValueMode
		param := editState.Parameters[editState.SelectedIndex]
		editState.EditingValue = param.Value
		editState.CursorPos = len(param.Value)
	case panels.QueryEditValueMode:
		// Save changes
		m.currentReq.UpdateQueryParameter(editState.SelectedIndex, editState.EditingKey, editState.EditingValue)
		editState.Mode = panels.QueryViewMode
		editState.EditingKey = ""
		editState.EditingValue = ""
		editState.OriginalKey = ""
		return *m, h.saveCurrentRequest(m)
	case panels.QueryAddMode:
		// Save new parameter
		if editState.EditingKey != "" {
			m.currentReq.AddQueryParameter(editState.EditingKey, editState.EditingValue)
			editState.Mode = panels.QueryViewMode
			editState.EditingKey = ""
			editState.EditingValue = ""
			editState.SelectedIndex = len(editState.Parameters) - 1
			return *m, h.saveCurrentRequest(m)
		}
	}
	return *m, nil
}

// handleQueryParameterUp handles up navigation in query parameters
func (h *InputHandler) handleQueryParameterUp(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeQueryEditState()
	editState := m.currentReq.QueryEditState
	
	if editState.Mode == panels.QueryViewMode && editState.SelectedIndex > 0 {
		editState.SelectedIndex--
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleQueryParameterDown handles down navigation in query parameters
func (h *InputHandler) handleQueryParameterDown(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeQueryEditState()
	editState := m.currentReq.QueryEditState
	
	if editState.Mode == panels.QueryViewMode && editState.SelectedIndex < len(editState.Parameters)-1 {
		editState.SelectedIndex++
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleQueryParameterAdd handles adding a new query parameter
func (h *InputHandler) handleQueryParameterAdd(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeQueryEditState()
	editState := m.currentReq.QueryEditState
	
	if editState.Mode == panels.QueryViewMode {
		editState.Mode = panels.QueryAddMode
		editState.EditingKey = ""
		editState.EditingValue = ""
		editState.CursorPos = 0
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleQueryParameterEdit handles editing a query parameter
func (h *InputHandler) handleQueryParameterEdit(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeQueryEditState()
	editState := m.currentReq.QueryEditState
	
	if editState.Mode == panels.QueryViewMode && len(editState.Parameters) > 0 && editState.SelectedIndex < len(editState.Parameters) {
		param := editState.Parameters[editState.SelectedIndex]
		editState.Mode = panels.QueryEditKeyMode
		editState.EditingKey = param.Key
		editState.OriginalKey = param.Key
		editState.CursorPos = len(param.Key)
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleQueryParameterDelete handles deleting a query parameter
func (h *InputHandler) handleQueryParameterDelete(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeQueryEditState()
	editState := m.currentReq.QueryEditState
	
	if editState.Mode == panels.QueryViewMode && len(editState.Parameters) > 0 && editState.SelectedIndex < len(editState.Parameters) {
		editState.PendingDeletion = editState.SelectedIndex
	}
	return *m, nil
}

// handleQueryParameterDeleteConfirm handles deletion confirmation
func (h *InputHandler) handleQueryParameterDeleteConfirm(m *model, confirm bool) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeQueryEditState()
	editState := m.currentReq.QueryEditState
	
	if editState.PendingDeletion >= 0 {
		if confirm {
			m.currentReq.DeleteQueryParameter(editState.PendingDeletion)
			editState.PendingDeletion = -1
			return *m, h.saveCurrentRequest(m)
		} else {
			editState.PendingDeletion = -1
		}
	}
	return *m, nil
}

// handleQueryParameterCancel handles canceling query parameter editing
func (h *InputHandler) handleQueryParameterCancel(m *model) (tea.Model, tea.Cmd) {
	if m.currentReq == nil || m.currentReq.QueryEditState == nil {
		return *m, nil
	}
	
	editState := m.currentReq.QueryEditState
	editState.Mode = panels.QueryViewMode
	editState.EditingKey = ""
	editState.EditingValue = ""
	editState.OriginalKey = ""
	editState.PendingDeletion = -1
	return *m, nil
}

// handleQueryParameterTextInput handles text input for query parameter editing
func (h *InputHandler) handleQueryParameterTextInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentReq == nil || m.currentReq.QueryEditState == nil {
		return *m, nil
	}
	
	editState := m.currentReq.QueryEditState
	
	// Only handle text input in editing modes
	if editState.Mode != panels.QueryEditKeyMode && editState.Mode != panels.QueryEditValueMode && editState.Mode != panels.QueryAddMode {
		return *m, nil
	}
	
	switch msg.String() {
	case "left":
		if editState.CursorPos > 0 {
			editState.CursorPos--
		}
	case "right":
		var currentText string
		if editState.Mode == panels.QueryEditKeyMode || editState.Mode == panels.QueryAddMode {
			currentText = editState.EditingKey
		} else {
			currentText = editState.EditingValue
		}
		if editState.CursorPos < len(currentText) {
			editState.CursorPos++
		}
	case "home":
		editState.CursorPos = 0
	case "end":
		var currentText string
		if editState.Mode == panels.QueryEditKeyMode || editState.Mode == panels.QueryAddMode {
			currentText = editState.EditingKey
		} else {
			currentText = editState.EditingValue
		}
		editState.CursorPos = len(currentText)
	case "backspace":
		if editState.CursorPos > 0 {
			if editState.Mode == panels.QueryEditKeyMode || editState.Mode == panels.QueryAddMode {
				editState.EditingKey = editState.EditingKey[:editState.CursorPos-1] + editState.EditingKey[editState.CursorPos:]
			} else {
				editState.EditingValue = editState.EditingValue[:editState.CursorPos-1] + editState.EditingValue[editState.CursorPos:]
			}
			editState.CursorPos--
		}
	case "delete":
		var currentText string
		if editState.Mode == panels.QueryEditKeyMode || editState.Mode == panels.QueryAddMode {
			currentText = editState.EditingKey
		} else {
			currentText = editState.EditingValue
		}
		if editState.CursorPos < len(currentText) {
			if editState.Mode == panels.QueryEditKeyMode || editState.Mode == panels.QueryAddMode {
				editState.EditingKey = editState.EditingKey[:editState.CursorPos] + editState.EditingKey[editState.CursorPos+1:]
			} else {
				editState.EditingValue = editState.EditingValue[:editState.CursorPos] + editState.EditingValue[editState.CursorPos+1:]
			}
		}
	case "tab":
		if editState.Mode == panels.QueryEditKeyMode {
			// Switch to editing value
			editState.Mode = panels.QueryEditValueMode
			param := editState.Parameters[editState.SelectedIndex]
			editState.EditingValue = param.Value
			editState.CursorPos = len(param.Value)
		} else if editState.Mode == panels.QueryAddMode {
			// In add mode, tab just moves cursor to value (both are shown)
			editState.CursorPos = 0
		}
	case "shift+tab":
		if editState.Mode == panels.QueryEditValueMode {
			// Switch back to editing key
			editState.Mode = panels.QueryEditKeyMode
			editState.CursorPos = len(editState.EditingKey)
		}
	default:
		// Handle regular character input
		if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
			char := msg.String()
			if editState.Mode == panels.QueryEditKeyMode || editState.Mode == panels.QueryAddMode {
				editState.EditingKey = editState.EditingKey[:editState.CursorPos] + char + editState.EditingKey[editState.CursorPos:]
			} else {
				editState.EditingValue = editState.EditingValue[:editState.CursorPos] + char + editState.EditingValue[editState.CursorPos:]
			}
			editState.CursorPos++
		}
	}
	
	return *m, nil
}

// saveCurrentRequest syncs the current request data in memory
func (h *InputHandler) saveCurrentRequest(m *model) tea.Cmd {
	if m.currentReq != nil {
		m.currentReq.SyncQueryToMap()
		m.currentReq.SyncTagsToSlice()
		m.currentReq.SyncHeadersToMap()
	}
	return nil
}

// Tag handling functions

// handleTagEnter handles enter key in tag editing
func (h *InputHandler) handleTagEnter(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeTagEditState()
	editState := m.currentReq.TagEditState
	
	switch editState.Mode {
	case panels.TagViewMode:
		// Start editing current tag
		if len(editState.Tags) > 0 && editState.SelectedIndex < len(editState.Tags) {
			tag := editState.Tags[editState.SelectedIndex]
			editState.Mode = panels.TagEditingMode
			editState.EditingTag = tag
			editState.CursorPos = len(tag)
		}
	case panels.TagEditingMode:
		// Save changes
		if editState.EditingTag != "" {
			m.currentReq.UpdateTag(editState.SelectedIndex, editState.EditingTag)
		}
		editState.Mode = panels.TagViewMode
		editState.EditingTag = ""
		return *m, h.saveCurrentRequest(m)
	case panels.TagAddMode:
		// Save new tag
		if editState.EditingTag != "" {
			m.currentReq.AddTag(editState.EditingTag)
			editState.Mode = panels.TagViewMode
			editState.EditingTag = ""
			editState.SelectedIndex = len(editState.Tags) - 1
			return *m, h.saveCurrentRequest(m)
		}
	}
	return *m, nil
}

// handleTagUp handles up navigation in tags
func (h *InputHandler) handleTagUp(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeTagEditState()
	editState := m.currentReq.TagEditState
	
	if editState.Mode == panels.TagViewMode && editState.SelectedIndex > 0 {
		editState.SelectedIndex--
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleTagDown handles down navigation in tags
func (h *InputHandler) handleTagDown(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeTagEditState()
	editState := m.currentReq.TagEditState
	
	if editState.Mode == panels.TagViewMode && editState.SelectedIndex < len(editState.Tags)-1 {
		editState.SelectedIndex++
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleTagAdd handles adding a new tag
func (h *InputHandler) handleTagAdd(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeTagEditState()
	editState := m.currentReq.TagEditState
	
	if editState.Mode == panels.TagViewMode {
		editState.Mode = panels.TagAddMode
		editState.EditingTag = ""
		editState.CursorPos = 0
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleTagEdit handles editing a tag
func (h *InputHandler) handleTagEdit(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeTagEditState()
	editState := m.currentReq.TagEditState
	
	if editState.Mode == panels.TagViewMode && len(editState.Tags) > 0 && editState.SelectedIndex < len(editState.Tags) {
		tag := editState.Tags[editState.SelectedIndex]
		editState.Mode = panels.TagEditingMode
		editState.EditingTag = tag
		editState.CursorPos = len(tag)
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleTagDelete handles deleting a tag
func (h *InputHandler) handleTagDelete(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeTagEditState()
	editState := m.currentReq.TagEditState
	
	if editState.Mode == panels.TagViewMode && len(editState.Tags) > 0 && editState.SelectedIndex < len(editState.Tags) {
		editState.PendingDeletion = editState.SelectedIndex
	}
	return *m, nil
}

// handleTagDeleteConfirm handles deletion confirmation
func (h *InputHandler) handleTagDeleteConfirm(m *model, confirm bool) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeTagEditState()
	editState := m.currentReq.TagEditState
	
	if editState.PendingDeletion >= 0 {
		if confirm {
			m.currentReq.DeleteTag(editState.PendingDeletion)
			editState.PendingDeletion = -1
			return *m, h.saveCurrentRequest(m)
		} else {
			editState.PendingDeletion = -1
		}
	}
	return *m, nil
}

// handleTagCancel handles canceling tag editing
func (h *InputHandler) handleTagCancel(m *model) (tea.Model, tea.Cmd) {
	if m.currentReq == nil || m.currentReq.TagEditState == nil {
		return *m, nil
	}
	
	editState := m.currentReq.TagEditState
	editState.Mode = panels.TagViewMode
	editState.EditingTag = ""
	editState.PendingDeletion = -1
	return *m, nil
}

// handleTagTextInput handles text input for tag editing
func (h *InputHandler) handleTagTextInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentReq == nil || m.currentReq.TagEditState == nil {
		return *m, nil
	}
	
	editState := m.currentReq.TagEditState
	
	// Only handle text input in editing modes
	if editState.Mode != panels.TagEditingMode && editState.Mode != panels.TagAddMode {
		return *m, nil
	}
	
	switch msg.String() {
	case "left":
		if editState.CursorPos > 0 {
			editState.CursorPos--
		}
	case "right":
		if editState.CursorPos < len(editState.EditingTag) {
			editState.CursorPos++
		}
	case "home":
		editState.CursorPos = 0
	case "end":
		editState.CursorPos = len(editState.EditingTag)
	case "backspace":
		if editState.CursorPos > 0 {
			editState.EditingTag = editState.EditingTag[:editState.CursorPos-1] + editState.EditingTag[editState.CursorPos:]
			editState.CursorPos--
		}
	case "delete":
		if editState.CursorPos < len(editState.EditingTag) {
			editState.EditingTag = editState.EditingTag[:editState.CursorPos] + editState.EditingTag[editState.CursorPos+1:]
		}
	default:
		// Handle regular character input
		if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
			char := msg.String()
			editState.EditingTag = editState.EditingTag[:editState.CursorPos] + char + editState.EditingTag[editState.CursorPos:]
			editState.CursorPos++
		}
	}
	
	return *m, nil
}

// Header handling functions

// handleHeaderEnter handles enter key in header editing
func (h *InputHandler) handleHeaderEnter(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeHeaderEditState()
	editState := m.currentReq.HeaderEditState
	
	switch editState.Mode {
	case panels.HeaderViewMode:
		// Start editing current header key
		if len(editState.Parameters) > 0 && editState.SelectedIndex < len(editState.Parameters) {
			param := editState.Parameters[editState.SelectedIndex]
			editState.Mode = panels.HeaderEditKeyMode
			editState.EditingKey = param.Key
			editState.OriginalKey = param.Key
			editState.CursorPos = len(param.Key)
		}
	case panels.HeaderEditKeyMode:
		// Move to editing value
		editState.Mode = panels.HeaderEditValueMode
		param := editState.Parameters[editState.SelectedIndex]
		editState.EditingValue = param.Value
		editState.CursorPos = len(param.Value)
	case panels.HeaderEditValueMode:
		// Save changes
		m.currentReq.UpdateHeader(editState.SelectedIndex, editState.EditingKey, editState.EditingValue)
		editState.Mode = panels.HeaderViewMode
		editState.EditingKey = ""
		editState.EditingValue = ""
		editState.OriginalKey = ""
		return *m, h.saveCurrentRequest(m)
	case panels.HeaderAddMode:
		// Save new header
		if editState.EditingKey != "" {
			m.currentReq.AddHeader(editState.EditingKey, editState.EditingValue)
			editState.Mode = panels.HeaderViewMode
			editState.EditingKey = ""
			editState.EditingValue = ""
			editState.SelectedIndex = len(editState.Parameters) - 1
			return *m, h.saveCurrentRequest(m)
		}
	}
	return *m, nil
}

// handleHeaderUp handles up navigation in headers
func (h *InputHandler) handleHeaderUp(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeHeaderEditState()
	editState := m.currentReq.HeaderEditState
	
	if editState.Mode == panels.HeaderViewMode && editState.SelectedIndex > 0 {
		editState.SelectedIndex--
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleHeaderDown handles down navigation in headers
func (h *InputHandler) handleHeaderDown(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeHeaderEditState()
	editState := m.currentReq.HeaderEditState
	
	if editState.Mode == panels.HeaderViewMode && editState.SelectedIndex < len(editState.Parameters)-1 {
		editState.SelectedIndex++
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleHeaderAdd handles adding a new header
func (h *InputHandler) handleHeaderAdd(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeHeaderEditState()
	editState := m.currentReq.HeaderEditState
	
	if editState.Mode == panels.HeaderViewMode {
		editState.Mode = panels.HeaderAddMode
		editState.EditingKey = ""
		editState.EditingValue = ""
		editState.CursorPos = 0
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleHeaderEdit handles editing a header
func (h *InputHandler) handleHeaderEdit(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeHeaderEditState()
	editState := m.currentReq.HeaderEditState
	
	if editState.Mode == panels.HeaderViewMode && len(editState.Parameters) > 0 && editState.SelectedIndex < len(editState.Parameters) {
		param := editState.Parameters[editState.SelectedIndex]
		editState.Mode = panels.HeaderEditKeyMode
		editState.EditingKey = param.Key
		editState.OriginalKey = param.Key
		editState.CursorPos = len(param.Key)
		editState.PendingDeletion = -1
	}
	return *m, nil
}

// handleHeaderDelete handles deleting a header
func (h *InputHandler) handleHeaderDelete(m *model) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeHeaderEditState()
	editState := m.currentReq.HeaderEditState
	
	if editState.Mode == panels.HeaderViewMode && len(editState.Parameters) > 0 && editState.SelectedIndex < len(editState.Parameters) {
		editState.PendingDeletion = editState.SelectedIndex
	}
	return *m, nil
}

// handleHeaderDeleteConfirm handles deletion confirmation
func (h *InputHandler) handleHeaderDeleteConfirm(m *model, confirm bool) (tea.Model, tea.Cmd) {
	m.currentReq.InitializeHeaderEditState()
	editState := m.currentReq.HeaderEditState
	
	if editState.PendingDeletion >= 0 {
		if confirm {
			m.currentReq.DeleteHeader(editState.PendingDeletion)
			editState.PendingDeletion = -1
			return *m, h.saveCurrentRequest(m)
		} else {
			editState.PendingDeletion = -1
		}
	}
	return *m, nil
}

// handleHeaderCancel handles canceling header editing
func (h *InputHandler) handleHeaderCancel(m *model) (tea.Model, tea.Cmd) {
	if m.currentReq == nil || m.currentReq.HeaderEditState == nil {
		return *m, nil
	}
	
	editState := m.currentReq.HeaderEditState
	editState.Mode = panels.HeaderViewMode
	editState.EditingKey = ""
	editState.EditingValue = ""
	editState.OriginalKey = ""
	editState.PendingDeletion = -1
	return *m, nil
}

// handleHeaderTextInput handles text input for header editing
func (h *InputHandler) handleHeaderTextInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentReq == nil || m.currentReq.HeaderEditState == nil {
		return *m, nil
	}
	
	editState := m.currentReq.HeaderEditState
	
	// Only handle text input in editing modes
	if editState.Mode != panels.HeaderEditKeyMode && editState.Mode != panels.HeaderEditValueMode && editState.Mode != panels.HeaderAddMode {
		return *m, nil
	}
	
	switch msg.String() {
	case "left":
		if editState.CursorPos > 0 {
			editState.CursorPos--
		}
	case "right":
		var currentText string
		if editState.Mode == panels.HeaderEditKeyMode || editState.Mode == panels.HeaderAddMode {
			currentText = editState.EditingKey
		} else {
			currentText = editState.EditingValue
		}
		if editState.CursorPos < len(currentText) {
			editState.CursorPos++
		}
	case "home":
		editState.CursorPos = 0
	case "end":
		var currentText string
		if editState.Mode == panels.HeaderEditKeyMode || editState.Mode == panels.HeaderAddMode {
			currentText = editState.EditingKey
		} else {
			currentText = editState.EditingValue
		}
		editState.CursorPos = len(currentText)
	case "backspace":
		if editState.CursorPos > 0 {
			if editState.Mode == panels.HeaderEditKeyMode || editState.Mode == panels.HeaderAddMode {
				editState.EditingKey = editState.EditingKey[:editState.CursorPos-1] + editState.EditingKey[editState.CursorPos:]
			} else {
				editState.EditingValue = editState.EditingValue[:editState.CursorPos-1] + editState.EditingValue[editState.CursorPos:]
			}
			editState.CursorPos--
		}
	case "delete":
		var currentText string
		if editState.Mode == panels.HeaderEditKeyMode || editState.Mode == panels.HeaderAddMode {
			currentText = editState.EditingKey
		} else {
			currentText = editState.EditingValue
		}
		if editState.CursorPos < len(currentText) {
			if editState.Mode == panels.HeaderEditKeyMode || editState.Mode == panels.HeaderAddMode {
				editState.EditingKey = editState.EditingKey[:editState.CursorPos] + editState.EditingKey[editState.CursorPos+1:]
			} else {
				editState.EditingValue = editState.EditingValue[:editState.CursorPos] + editState.EditingValue[editState.CursorPos+1:]
			}
		}
	case "tab":
		if editState.Mode == panels.HeaderEditKeyMode {
			// Switch to editing value
			editState.Mode = panels.HeaderEditValueMode
			param := editState.Parameters[editState.SelectedIndex]
			editState.EditingValue = param.Value
			editState.CursorPos = len(param.Value)
		} else if editState.Mode == panels.HeaderAddMode {
			// In add mode, tab just moves cursor to value (both are shown)
			editState.CursorPos = 0
		}
	case "shift+tab":
		if editState.Mode == panels.HeaderEditValueMode {
			// Switch back to editing key
			editState.Mode = panels.HeaderEditKeyMode
			editState.CursorPos = len(editState.EditingKey)
		}
	default:
		// Handle regular character input
		if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
			char := msg.String()
			if editState.Mode == panels.HeaderEditKeyMode || editState.Mode == panels.HeaderAddMode {
				editState.EditingKey = editState.EditingKey[:editState.CursorPos] + char + editState.EditingKey[editState.CursorPos:]
			} else {
				editState.EditingValue = editState.EditingValue[:editState.CursorPos] + char + editState.EditingValue[editState.CursorPos:]
			}
			editState.CursorPos++
		}
	}
	
	return *m, nil
}