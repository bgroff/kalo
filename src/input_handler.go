package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	request "kalo/src/panels/request"
)

type InputHandler struct {
	collectionsHandler    *CollectionsInputHandler
	requestHandler        *RequestInputHandler
	responseHandler       *ResponseInputHandler
	commandPaletteHandler *CommandPaletteInputHandler
}

func NewInputHandler() *InputHandler {
	return &InputHandler{
		collectionsHandler:    NewCollectionsInputHandler(),
		requestHandler:        NewRequestInputHandler(),
		responseHandler:       NewResponseInputHandler(),
		commandPaletteHandler: NewCommandPaletteInputHandler(),
	}
}

func (h *InputHandler) HandleKeyboardInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle input dialog first (highest priority)
	if m.inputDialog.IsVisible() {
		return h.handleDialogInput(m, msg)
	}

	// Handle command palette (has highest priority when active)
	if h.commandPaletteHandler.CanHandleInput(m) {
		newModel, cmd := h.commandPaletteHandler.HandleInput(msg, m)
		return newModel, cmd
	}

	// Handle global shortcuts before delegating to panels
	switch msg.Type {
	case tea.KeyTab:
		// Check if we're in filter mode - if so, let filter handler handle Tab for autocomplete
		if m.filterMode() {
			return h.handleFilterInput(m, msg)
		}
		// Check if request panel is in edit mode - if so, let it handle Tab
		if m.activePanel == requestPanel && h.requestHandler.IsInEditMode(m) {
			return h.handleRequestInput(m, msg)
		}
		return h.handleTabNavigation(m, msg)
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlP:
		// Open command palette
		m.commandPalette.Show()
		return m, nil
	}

	// Delegate to the active panel's input handler
	switch m.activePanel {
	case collectionsPanel:
		return h.handleCollectionsInput(m, msg)
	case requestPanel:
		return h.handleRequestInput(m, msg)
	case responsePanel:
		return h.handleResponseInput(m, msg)
	}

	return m, nil
}

// handleTabNavigation cycles through panels
func (h *InputHandler) handleTabNavigation(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case collectionsPanel:
		m.activePanel = requestPanel
	case requestPanel:
		m.activePanel = responsePanel
	case responsePanel:
		// If command palette exists and is available, go there, otherwise back to collections
		m.activePanel = collectionsPanel
	}
	return m, nil
}

// handleCollectionsInput delegates to collections panel handler
func (h *InputHandler) handleCollectionsInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newModel, cmd := h.collectionsHandler.HandleInput(msg, m)
	return newModel, cmd
}

// handleRequestInput delegates to request panel handler
func (h *InputHandler) handleRequestInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newModel, cmd := h.requestHandler.HandleInput(msg, m)
	return newModel, cmd
}

// handleResponseInput delegates to response panel handler
func (h *InputHandler) handleResponseInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check if response handler can handle JQ filter input
	if m.filterMode() && m.filterType() == JQFilter {
		// Let main handler handle JQ filter for now since it's complex
		return h.handleFilterInput(m, msg)
	}

	newModel, cmd := h.responseHandler.HandleInput(msg, m)
	return newModel, cmd
}

// GetFooterText returns footer text for the active panel
func (h *InputHandler) GetFooterText(m *model) string {
	// Handle special states first
	if m.isLoading {
		return "Loading... • Tab: Switch panels • Ctrl+P: Command palette • Ctrl+C: Quit"
	}

	// Command palette takes priority
	if h.commandPaletteHandler.IsInEditMode(m) {
		return h.commandPaletteHandler.GetFooterText(m)
	}

	// Handle filter modes with special display logic
	if m.filterMode() {
		if m.filterType() == CollectionsFilter {
			return h.collectionsHandler.GetFooterText(m)
		}
		return h.getFilterFooterText(m)
	}

	// Check panel-specific footer text
	switch m.activePanel {
	case collectionsPanel:
		return h.collectionsHandler.GetFooterText(m)
	case requestPanel:
		return h.requestHandler.GetFooterText(m)
	case responsePanel:
		return h.responseHandler.GetFooterText(m)
	}

	return "Tab: switch panels | Ctrl+P: command palette | Ctrl+C: quit"
}

// Temporary direct implementations (will be moved to panel handlers)

// Footer text helpers

func (h *InputHandler) getFilterFooterText(m *model) string {
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
			inputWithCursor := renderFilterCursor(m.filterInput(), m.filterCursorPos())
			return fmt.Sprintf("%s filter: %s • ↑↓: Navigate • Tab: Complete • Enter: Apply • Esc: Cancel | %s", filterName, inputWithCursor, suggestionText)
		}
	}

	// Show input with cursor (for both jq without suggestions and collections filter)
	inputWithCursor := renderFilterCursor(m.filterInput(), m.filterCursorPos())
	return fmt.Sprintf("%s filter: %s • Enter: Apply • Esc: Cancel", filterName, inputWithCursor)
}

// Helper methods that still need to be moved to appropriate panel handlers

// isInTextInputMode checks if we're currently in a text input editing mode
func (h *InputHandler) isInTextInputMode(m *model) bool {
	// Check for query parameter text input mode
	if m.activePanel == requestPanel && m.requestCursor == request.QuerySection && m.currentReq != nil {
		if m.currentReq.QueryEditState != nil {
			mode := m.currentReq.QueryEditState.Mode
			return mode == request.QueryEditKeyMode || mode == request.QueryEditValueMode || mode == request.QueryAddMode
		}
	}

	// Check for header text input mode
	if m.activePanel == requestPanel && m.requestCursor == request.HeadersSection && m.currentReq != nil {
		if m.currentReq.HeaderEditState != nil {
			mode := m.currentReq.HeaderEditState.Mode
			return mode == request.HeaderEditKeyMode || mode == request.HeaderEditValueMode || mode == request.HeaderAddMode
		}
	}

	return false
}

// handleTextInputMode handles input when in a text input editing mode
func (h *InputHandler) handleTextInputMode(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Only allow essential navigation keys to interrupt text input mode
	switch msg.Type {
	case tea.KeyEsc:
		// Always allow escape to exit text input mode
		return h.handleTextInputEscape(m, msg)
	case tea.KeyTab:
		// Allow tab for panel switching even in text input mode
		return h.handleTabNavigation(m, msg)
	}

	// Delegate to appropriate text input handler based on current section
	if m.requestCursor == request.QuerySection && m.currentReq != nil {
		return h.handleQueryParameterTextInput(m, msg)
	}
	if m.requestCursor == request.HeadersSection && m.currentReq != nil {
		return h.handleHeaderTextInput(m, msg)
	}

	return m, nil
}

// handleFilterInput handles input when in filter mode
func (h *InputHandler) handleFilterInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.exitFilter()
		m.updateCollectionsViewport()
		return m, nil
	case tea.KeyEnter:
		// For collections filter, apply and exit filter mode but keep filtered results
		if m.filterType() == CollectionsFilter {
			m.setLastCollectionsFilter(m.filterInput())
			m.setFilterMode(false)
			return m, nil
		}
		// For jq filter, apply the filter
		if m.filterType() == JQFilter {
			return m, m.applyFilter()
		}
		// Don't call m.exitFilter() as that would restore original collections
		return m, nil
	case tea.KeyTab:
		// Handle Tab for JQ filter autocomplete
		if m.filterType() == JQFilter && m.showSuggestions() {
			filteredSuggestions := m.getFilteredSuggestions()
			if len(filteredSuggestions) > 0 {
				selectedIndex := m.selectedSuggestion()
				if selectedIndex >= 0 && selectedIndex < len(filteredSuggestions) {
					// Complete with selected suggestion
					selectedSuggestion := filteredSuggestions[selectedIndex]
					input := m.filterInput()
					
					// Find the last word boundary to replace
					pos := m.filterCursorPos()
					wordStart := pos
					for wordStart > 0 && input[wordStart-1] != ' ' && input[wordStart-1] != '.' && input[wordStart-1] != '[' && input[wordStart-1] != '(' {
						wordStart--
					}
					
					// Replace the partial word with the suggestion
					newInput := input[:wordStart] + selectedSuggestion + input[pos:]
					m.setFilterInput(newInput)
					m.setFilterCursorPos(wordStart + len(selectedSuggestion))
				}
			}
		}
		return m, nil
	case tea.KeyUp:
		// Navigate suggestions up for JQ filter
		if m.filterType() == JQFilter && m.showSuggestions() {
			filteredSuggestions := m.getFilteredSuggestions()
			if len(filteredSuggestions) > 0 {
				selectedIndex := m.selectedSuggestion()
				if selectedIndex > 0 {
					m.setSelectedSuggestion(selectedIndex - 1)
				} else {
					m.setSelectedSuggestion(len(filteredSuggestions) - 1) // Wrap to last
				}
			}
		}
		return m, nil
	case tea.KeyDown:
		// Navigate suggestions down for JQ filter
		if m.filterType() == JQFilter && m.showSuggestions() {
			filteredSuggestions := m.getFilteredSuggestions()
			if len(filteredSuggestions) > 0 {
				selectedIndex := m.selectedSuggestion()
				if selectedIndex < len(filteredSuggestions)-1 {
					m.setSelectedSuggestion(selectedIndex + 1)
				} else {
					m.setSelectedSuggestion(0) // Wrap to first
				}
			}
		}
		return m, nil
	case tea.KeyBackspace:
		if len(m.filterInput()) > 0 && m.filterCursorPos() > 0 {
			input := m.filterInput()
			pos := m.filterCursorPos()
			newInput := input[:pos-1] + input[pos:]
			m.setFilterInput(newInput)
			m.setFilterCursorPos(pos - 1)

			// Apply collections filter in real-time
			if m.filterType() == CollectionsFilter {
				m.applyCollectionsFilterResult()
				m.updateCollectionsViewport()
			}
		}
		return m, nil
	case tea.KeyLeft:
		if m.filterCursorPos() > 0 {
			m.setFilterCursorPos(m.filterCursorPos() - 1)
		}
		return m, nil
	case tea.KeyRight:
		// Apply collections filter in real-time
		if m.filterCursorPos() < len(m.filterInput()) {
			m.setFilterCursorPos(m.filterCursorPos() + 1)
		}
		return m, nil
	case tea.KeyHome, tea.KeyCtrlA:
		m.setFilterCursorPos(0)
		return m, nil
	case tea.KeyEnd, tea.KeyCtrlE:
		// Apply collections filter in real-time
		m.setFilterCursorPos(len(m.filterInput()))
		return m, nil
	case tea.KeyCtrlW:
		// Delete word backwards
		input := m.filterInput()
		pos := m.filterCursorPos()
		if pos > 0 {
			wordStart := m.findPreviousWordBoundary(input, pos)
			newInput := input[:wordStart] + input[pos:]
			m.setFilterInput(newInput)
			m.setFilterCursorPos(wordStart)

			if m.filterType() == CollectionsFilter {
				m.applyCollectionsFilterResult()
				m.updateCollectionsViewport()
			}
		}
		return m, nil
	case tea.KeyRunes:
		// Handle regular character input
		if len(msg.Runes) == 1 && msg.Runes[0] >= ' ' && msg.Runes[0] <= '~' {
			input := m.filterInput()
			pos := m.filterCursorPos()
			newInput := input[:pos] + string(msg.Runes[0]) + input[pos:]
			m.setFilterInput(newInput)
			m.setFilterCursorPos(pos + 1)

			if m.filterType() == CollectionsFilter {
				m.applyCollectionsFilterResult()
				m.updateCollectionsViewport()
			}
		}
		return m, nil
	}

	return m, nil
}

// handleDialogInput handles input when input dialog is visible
func (h *InputHandler) handleDialogInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.inputDialog.Hide()
		return m, nil
	case tea.KeyEnter:
		m.inputDialog.Confirm()
		input, action, actionData, _ := m.inputDialog.GetResult()
		m.inputDialog.Hide()
		return m, m.executeInputCommand(action, input, actionData)
	case tea.KeyTab:
		m.inputDialog.SwitchField()
		return m, nil
	case tea.KeyUp:
		m.inputDialog.MoveMethodSelection(-1)
		return m, nil
	case tea.KeyDown:
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

// Placeholder methods - these need full implementations from the original file
func (h *InputHandler) handleTextInputEscape(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// TODO: Implement proper escape handling for different text input modes
	return m, nil
}

func (h *InputHandler) handleQueryParameterTextInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// TODO: Move query parameter text input logic here
	return m, nil
}

func (h *InputHandler) handleTagTextInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// TODO: Move tag text input logic here
	return m, nil
}

func (h *InputHandler) handleHeaderTextInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// TODO: Move header text input logic here
	return m, nil
}

func (h *InputHandler) handleRequestSectionAction(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// TODO: Move request section action logic here
	return m, nil
}
