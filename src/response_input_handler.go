package main

import (
	tea "github.com/charmbracelet/bubbletea"
	response "kalo/src/panels/response"
)

// ResponseInputHandler handles input for the response panel
type ResponseInputHandler struct{}

// NewResponseInputHandler creates a new response input handler
func NewResponseInputHandler() *ResponseInputHandler {
	return &ResponseInputHandler{}
}

// HandleInput processes key input for the response panel
func (h *ResponseInputHandler) HandleInput(key tea.KeyMsg, m *model) (*model, tea.Cmd) {
	// Handle jq filter mode
	if m.filterMode() && m.filterType() == JQFilter {
		return h.handleJQFilterInput(m, key)
	}

	switch key.Type {
	case tea.KeyLeft:
		if m.responseActiveTab > 0 {
			m.responseActiveTab--
		} else {
			m.responseActiveTab = len(response.GetResponseTabNames()) - 1 // Wrap to last tab
		}
		return m, nil
	case tea.KeyRight:
		maxTabs := len(response.GetResponseTabNames())
		if m.responseActiveTab < maxTabs-1 {
			m.responseActiveTab++
		} else {
			m.responseActiveTab = 0 // Wrap to first tab
		}
		return m, nil
	case tea.KeyUp:
		if m.responseCursor == response.ResponseHeadersSection {
			m.headersViewport.LineUp(1)
		}
		return m, nil
	case tea.KeyDown:
		if m.responseCursor == response.ResponseHeadersSection {
			m.headersViewport.LineDown(1)
		}
		return m, nil
	case tea.KeyCtrlD, tea.KeyPgDown:
		if m.responseCursor == response.ResponseHeadersSection {
			m.headersViewport.HalfViewDown()
		}
		return m, nil
	case tea.KeyCtrlU, tea.KeyPgUp:
		if m.responseCursor == response.ResponseHeadersSection {
			m.headersViewport.HalfViewUp()
		}
		return m, nil
	case tea.KeyHome:
		if m.responseCursor == response.ResponseHeadersSection {
			m.headersViewport.GotoTop()
		}
		return m, nil
	case tea.KeyEnd:
		if m.responseCursor == response.ResponseHeadersSection {
			m.headersViewport.GotoBottom()
		}
		return m, nil
	case tea.KeyCtrlR:
		// Reset jq filter
		if m.appliedJQFilter() != "" {
			m.responseViewport.SetContent(m.originalResponse)
			m.setAppliedJQFilter("")
			m.filterManager.Reset(JQFilter)
		}
		return m, nil
	case tea.KeyRunes:
		// Handle character-based shortcuts
		switch string(key.Runes) {
		case "h":
			if m.responseActiveTab > 0 {
				m.responseActiveTab--
			} else {
				m.responseActiveTab = len(response.GetResponseTabNames()) - 1 // Wrap to last tab
			}
			return m, nil
		case "l":
			maxTabs := len(response.GetResponseTabNames())
			if m.responseActiveTab < maxTabs-1 {
				m.responseActiveTab++
			} else {
				m.responseActiveTab = 0 // Wrap to first tab
			}
			return m, nil
		case "k":
			if m.responseCursor == response.ResponseHeadersSection {
				m.headersViewport.LineUp(1)
			}
			return m, nil
		case "j":
			if m.responseCursor == response.ResponseHeadersSection {
				m.headersViewport.LineDown(1)
			}
			return m, nil
		case "g":
			if m.responseCursor == response.ResponseHeadersSection {
				m.headersViewport.GotoTop()
			}
			return m, nil
		case "G":
			if m.responseCursor == response.ResponseHeadersSection {
				m.headersViewport.GotoBottom()
			}
			return m, nil
		case "/":
			// Start jq filter
			m.responseCursor = response.ResponseBodySection
			m.startFilter(JQFilter)
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

// CanHandleInput returns true if response panel can handle input
func (h *ResponseInputHandler) CanHandleInput(m *model) bool {
	return m.activePanel == responsePanel
}

// GetFooterText returns footer text for response panel
func (h *ResponseInputHandler) GetFooterText(m *model) string {
	if m.appliedJQFilter() != "" {
		return "←/→: switch tabs | /: filter | Ctrl+R: reset filter | Tab: next panel (filtered)"
	}
	
	return "←/→: switch tabs | /: jq filter | Tab: next panel"
}

// IsInEditMode returns true if response panel is in edit mode
func (h *ResponseInputHandler) IsInEditMode(m *model) bool {
	return m.filterMode() && m.filterType() == JQFilter
}

// handleJQFilterInput handles jq filter mode input (delegates to main handler for now)
func (h *ResponseInputHandler) handleJQFilterInput(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	// This delegates to the main handler's handleFilterInput for JQ filters
	// The JQ filter logic is more complex and involves the filter manager
	// For now, we return unchanged - the main handler will handle JQ filters
	return m, nil
}