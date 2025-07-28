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

	// Handle command palette
	if _, selectedCmd, handled := m.commandPalette.HandleInput(msg); handled {
		return h.handleCommandPalette(m, selectedCmd)
	}

	// Normal navigation and shortcuts
	return h.handleNormalInput(m, msg)
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
			return *m, m.executeRequest()
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
		if m.activePanel == responsePanel && m.responseCursor == panels.ResponseBodySection {
			m.responseCursor = panels.ResponseHeadersSection
		}
	case "right":
		if m.activePanel == responsePanel && m.responseCursor == panels.ResponseHeadersSection {
			m.responseCursor = panels.ResponseBodySection
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
	}
	
	return *m, nil
}