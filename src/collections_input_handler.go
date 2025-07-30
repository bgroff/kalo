package main

import (
	tea "github.com/charmbracelet/bubbletea"
	collections "kalo/src/panels/collections"
	request "kalo/src/panels/request"
)

// CollectionsInputHandler handles input for the collections panel
type CollectionsInputHandler struct{}

// NewCollectionsInputHandler creates a new collections input handler
func NewCollectionsInputHandler() *CollectionsInputHandler {
	return &CollectionsInputHandler{}
}

// HandleInput processes key input for the collections panel
func (h *CollectionsInputHandler) HandleInput(key tea.KeyMsg, m *model) (*model, tea.Cmd) {
	// Handle filter mode input
	if m.filterMode() {
		return h.handleFilterInput(m, key)
	}

	switch key.Type {
	case tea.KeyUp:
		if m.selectedReq > 0 {
			// Find previous visible item
			for i := m.selectedReq - 1; i >= 0; i-- {
				if m.collections[i].IsVisible {
					m.selectedReq = i
					break
				}
			}
			m.updateCollectionsViewport()
		}
		return m, nil
	case tea.KeyDown:
		if m.selectedReq < len(m.collections)-1 {
			// Find next visible item
			for i := m.selectedReq + 1; i < len(m.collections); i++ {
				if m.collections[i].IsVisible {
					m.selectedReq = i
					break
				}
			}
			m.updateCollectionsViewport()
		}
		return m, nil
	case tea.KeyEnter, tea.KeySpace:
		if m.selectedReq >= 0 && m.selectedReq < len(m.collections) {
			item := m.collections[m.selectedReq]
			if item.IsFolder || item.IsTagGroup {
				m.toggleExpansion(m.selectedReq)
				m.updateCollectionsViewport()
			} else if item.RequestIndex >= 0 && item.RequestIndex < len(m.bruRequests) {
				// Execute the selected request
				m.currentReq = m.bruRequests[item.RequestIndex]
				m.requestCursor = request.QuerySection
				m.isLoading = true
				return m, tea.Batch(func() tea.Msg {
					resp, err := m.httpClient.ExecuteRequest(m.currentReq)
					return httpResponseMsg{response: resp, err: err}
				})
			}
		}
		return m, nil
	case tea.KeyCtrlD, tea.KeyPgDown:
		m.collectionsViewport.HalfViewDown()
		return m, nil
	case tea.KeyCtrlU, tea.KeyPgUp:
		m.collectionsViewport.HalfViewUp()
		return m, nil
	case tea.KeyHome:
		m.collectionsViewport.GotoTop()
		m.selectedReq = 0
		return m, nil
	case tea.KeyEnd:
		m.collectionsViewport.GotoBottom()
		if len(m.collections) > 0 {
			m.selectedReq = len(m.collections) - 1
		}
		return m, nil
	case tea.KeyCtrlR:
		if len(m.originalCollections()) > 0 {
			m.collections = make([]collections.CollectionItem, len(m.originalCollections()))
			copy(m.collections, m.originalCollections())
			m.setOriginalCollections(nil)
			m.selectedReq = 0
			m.filterManager.Reset(CollectionsFilter)
			m.updateVisibility()
			m.updateCollectionsViewport()
		}
		return m, nil
	case tea.KeyRunes:
		// Handle character-based shortcuts
		switch string(key.Runes) {
		case "k":
			if m.selectedReq > 0 {
				// Find previous visible item
				for i := m.selectedReq - 1; i >= 0; i-- {
					if m.collections[i].IsVisible {
						m.selectedReq = i
						break
					}
				}
				m.updateCollectionsViewport()
			}
			return m, nil
		case "j":
			if m.selectedReq < len(m.collections)-1 {
				// Find next visible item
				for i := m.selectedReq + 1; i < len(m.collections); i++ {
					if m.collections[i].IsVisible {
						m.selectedReq = i
						break
					}
				}
				m.updateCollectionsViewport()
			}
			return m, nil
		case "g":
			m.collectionsViewport.GotoTop()
			m.selectedReq = 0
			return m, nil
		case "G":
			m.collectionsViewport.GotoBottom()
			if len(m.collections) > 0 {
				m.selectedReq = len(m.collections) - 1
			}
			return m, nil
		case "/":
			m.startFilter(CollectionsFilter)
			return m, nil
		}
		return m, nil
	}

	// Handle complex key combinations that aren't easily represented as key constants
	switch key.String() {
	case "ctrl+enter":
		if m.selectedReq >= 0 && m.selectedReq < len(m.collections) {
			item := m.collections[m.selectedReq]
			if item.RequestIndex >= 0 && item.RequestIndex < len(m.bruRequests) {
				m.currentReq = m.bruRequests[item.RequestIndex]
				m.isLoading = true
				return m, tea.Batch(func() tea.Msg {
					resp, err := m.httpClient.ExecuteRequest(m.currentReq)
					return httpResponseMsg{response: resp, err: err}
				})
			}
		}
		return m, nil
	}

	return m, nil
}

// CanHandleInput returns true if collections panel can handle input
func (h *CollectionsInputHandler) CanHandleInput(m *model) bool {
	return m.activePanel == collectionsPanel
}

// GetFooterText returns footer text for collections panel
func (h *CollectionsInputHandler) GetFooterText(m *model) string {
	if len(m.originalCollections()) > 0 && len(m.collections) != len(m.originalCollections()) {
		return "↑/↓: navigate | Enter: expand/collapse | /: filter | Ctrl+R: reset filter | Tab: next panel"
	}
	
	return "↑/↓: navigate | Enter: expand/collapse | /: filter | Tab: next panel"
}

// IsInEditMode returns true if collections panel is in edit mode
func (h *CollectionsInputHandler) IsInEditMode(m *model) bool {
	return m.filterMode() && m.filterType() == CollectionsFilter
}

// handleFilterInput handles input when in filter mode (moved from main input handler)
func (h *CollectionsInputHandler) handleFilterInput(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
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
		if m.filterCursorPos() < len(m.filterInput()) {
			m.setFilterCursorPos(m.filterCursorPos() + 1)
		}
		return m, nil
	case tea.KeyHome, tea.KeyCtrlA:
		m.setFilterCursorPos(0)
		return m, nil
	case tea.KeyEnd, tea.KeyCtrlE:
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