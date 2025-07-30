package main

import (
	"encoding/json"
	"strings"
	
	tea "github.com/charmbracelet/bubbletea"
	request "kalo/src/panels/request"
)

// RequestInputHandler handles input for the request panel
type RequestInputHandler struct{}

// NewRequestInputHandler creates a new request input handler
func NewRequestInputHandler() *RequestInputHandler {
	return &RequestInputHandler{}
}

// HandleInput processes key input for the request panel
func (h *RequestInputHandler) HandleInput(key tea.KeyMsg, m *model) (*model, tea.Cmd) {
	// Handle text input modes first
	if h.isInTextInputMode(m) {
		return h.handleTextInputMode(m, key)
	}

	// Handle tab switching and navigation
	switch key.Type {
	case tea.KeyLeft:
		if m.requestActiveTab > 0 {
			m.requestActiveTab--
		} else {
			m.requestActiveTab = len(request.GetRequestTabNames()) - 1 // Wrap to last tab
		}
		m.requestCursor = request.GetRequestTabSection(m.requestActiveTab)
		return m, nil
	case tea.KeyRight:
		maxTabs := len(request.GetRequestTabNames())
		if m.requestActiveTab < maxTabs-1 {
			m.requestActiveTab++
		} else {
			m.requestActiveTab = 0 // Wrap to first tab
		}
		m.requestCursor = request.GetRequestTabSection(m.requestActiveTab)
		return m, nil
	case tea.KeyUp, tea.KeyDown:
		// Delegate to section-specific handlers first
		// They will handle intra-section navigation (e.g., between query parameters)
		// If they don't handle it, we'll fall back to section switching
		newModel, cmd := h.handleRequestSectionAction(m, key)
		if newModel != m || cmd != nil {
			// Section handler processed the key
			return newModel, cmd
		}
		
		// Fall back to section switching
		if key.Type == tea.KeyUp {
			maxSection := request.GetMaxRequestSection(m.currentReq)
			if m.requestCursor > 0 {
				m.requestCursor--
			} else {
				m.requestCursor = maxSection
			}
		} else { // tea.KeyDown
			maxSection := request.GetMaxRequestSection(m.currentReq)
			if m.requestCursor < maxSection {
				m.requestCursor++
			} else {
				m.requestCursor = 0
			}
		}
		return m, nil
	case tea.KeyEnter:
		// Delegate to section-specific handlers
		return h.handleRequestSectionAction(m, key)
	case tea.KeyRunes:
		// Handle character-based shortcuts
		switch string(key.Runes) {
		case "h":
			if m.requestActiveTab > 0 {
				m.requestActiveTab--
			} else {
				m.requestActiveTab = len(request.GetRequestTabNames()) - 1 // Wrap to last tab
			}
			m.requestCursor = request.GetRequestTabSection(m.requestActiveTab)
			return m, nil
		case "l":
			maxTabs := len(request.GetRequestTabNames())
			if m.requestActiveTab < maxTabs-1 {
				m.requestActiveTab++
			} else {
				m.requestActiveTab = 0 // Wrap to first tab
			}
			m.requestCursor = request.GetRequestTabSection(m.requestActiveTab)
			return m, nil
		case "k":
			maxSection := request.GetMaxRequestSection(m.currentReq)
			if m.requestCursor > 0 {
				m.requestCursor--
			} else {
				m.requestCursor = maxSection
			}
			return m, nil
		case "j":
			maxSection := request.GetMaxRequestSection(m.currentReq)
			if m.requestCursor < maxSection {
				m.requestCursor++
			} else {
				m.requestCursor = 0
			}
			return m, nil
		default:
			// Delegate other character keys to section handlers
			return h.handleRequestSectionAction(m, key)
		}
		return m, nil
	}

	return m, nil
}

// CanHandleInput returns true if request panel can handle input
func (h *RequestInputHandler) CanHandleInput(m *model) bool {
	return m.activePanel == requestPanel
}

// GetFooterText returns footer text for request panel
func (h *RequestInputHandler) GetFooterText(m *model) string {
	if h.isInTextInputMode(m) {
		// Check if we're in query or header edit mode for specific Tab behavior
		if (m.requestCursor == request.QuerySection && m.currentReq != nil && m.currentReq.QueryEditState != nil) ||
		   (m.requestCursor == request.HeadersSection && m.currentReq != nil && m.currentReq.HeaderEditState != nil) {
			return "Type to edit | Enter: confirm | Esc: cancel | Tab: key/value"
		}
		return "Type to edit | Enter: confirm | Esc: cancel"
	}
	return "←/→: switch tabs | ↑/↓: navigate | Enter: edit | a: add | d: delete | Tab: next panel"
}

// IsInEditMode returns true if request panel is in edit mode
func (h *RequestInputHandler) IsInEditMode(m *model) bool {
	return h.isInTextInputMode(m)
}

// isInTextInputMode checks if we're currently in a text input editing mode
func (h *RequestInputHandler) isInTextInputMode(m *model) bool {
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
	
	// Check for auth text input mode
	if m.activePanel == requestPanel && m.requestCursor == request.AuthSection && m.currentReq != nil {
		if m.currentReq.AuthEditState != nil {
			mode := m.currentReq.AuthEditState.Mode
			return mode == request.AuthTypeSelectMode || mode == request.AuthBearerEditMode || 
				   mode == request.AuthBasicUsernameEditMode || mode == request.AuthBasicPasswordEditMode
		}
	}
	
	// Check for body text input mode
	if m.activePanel == requestPanel && m.requestCursor == request.BodySection && m.currentReq != nil {
		if m.currentReq.BodyEditState != nil {
			return m.currentReq.BodyEditState.Mode == request.BodyTextEditMode
		}
	}
	
	return false
}

// handleTextInputMode handles input when in a text input editing mode
func (h *RequestInputHandler) handleTextInputMode(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	// Only allow essential navigation keys to interrupt text input mode
	switch msg.Type {
	case tea.KeyEsc:
		// Always allow escape to exit text input mode
		return h.handleTextInputEscape(m, msg)
	case tea.KeyTab:
		// In edit mode, Tab should switch between key/value, not switch panels
		// Delegate to appropriate text input handler to handle Tab switching
		// Only allow panel switching if we're not in an active edit mode
		if m.requestCursor == request.QuerySection && m.currentReq != nil {
			return h.handleQueryParameterTextInput(m, msg)
		}
		if m.requestCursor == request.HeadersSection && m.currentReq != nil {
			return h.handleHeaderTextInput(m, msg)
		}
		// For other sections, allow panel switching
		return m, nil
	}

	// Delegate to appropriate text input handler based on current section
	if m.requestCursor == request.QuerySection && m.currentReq != nil {
		return h.handleQueryParameterTextInput(m, msg)
	}
	if m.requestCursor == request.HeadersSection && m.currentReq != nil {
		return h.handleHeaderTextInput(m, msg)
	}
	if m.requestCursor == request.AuthSection && m.currentReq != nil {
		return h.handleAuthTextInput(m, msg)
	}
	if m.requestCursor == request.BodySection && m.currentReq != nil {
		return h.handleBodyTextInput(m, msg)
	}

	return m, nil
}

// Placeholder methods that need full implementations from the original file
func (h *RequestInputHandler) handleTextInputEscape(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	// TODO: Implement proper escape handling for different text input modes
	return m, nil
}

func (h *RequestInputHandler) handleQueryParameterTextInput(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	if m.currentReq == nil || m.currentReq.QueryEditState == nil {
		return m, nil
	}

	editState := m.currentReq.QueryEditState

	switch msg.Type {
	case tea.KeyEsc:
		// Exit edit mode
		editState.Mode = request.QueryViewMode
		editState.EditingKey = ""
		editState.EditingValue = ""
		editState.CursorPos = 0
		return m, nil

	case tea.KeyEnter:
		// Save the current edit
		switch editState.Mode {
		case request.QueryEditKeyMode:
			// Save key edit
			if editState.SelectedIndex < len(editState.Parameters) {
				editState.Parameters[editState.SelectedIndex].Key = editState.EditingKey
				m.currentReq.SyncQueryToMap()
			}
			editState.Mode = request.QueryViewMode
			editState.EditingKey = ""
			editState.CursorPos = 0
		case request.QueryEditValueMode:
			// Save value edit
			if editState.SelectedIndex < len(editState.Parameters) {
				editState.Parameters[editState.SelectedIndex].Value = editState.EditingValue
				m.currentReq.SyncQueryToMap()
			}
			editState.Mode = request.QueryViewMode
			editState.EditingValue = ""
			editState.CursorPos = 0
		case request.QueryAddMode:
			// Add new parameter
			if editState.EditingKey != "" {
				newParam := request.QueryParameter{
					Key:   editState.EditingKey,
					Value: editState.EditingValue,
				}
				editState.Parameters = append(editState.Parameters, newParam)
				m.currentReq.SyncQueryToMap()
				editState.Mode = request.QueryViewMode
				editState.EditingKey = ""
				editState.EditingValue = ""
				editState.CursorPos = 0
				editState.SelectedIndex = len(editState.Parameters) - 1
			}
		}
		return m, nil

	case tea.KeyTab:
		// Switch between key and value editing
		switch editState.Mode {
		case request.QueryEditKeyMode:
			// Switch to editing value
			editState.Mode = request.QueryEditValueMode
			editState.EditingValue = editState.Parameters[editState.SelectedIndex].Value
			editState.CursorPos = len(editState.EditingValue)
		case request.QueryEditValueMode:
			// Switch to editing key  
			editState.Mode = request.QueryEditKeyMode
			editState.EditingKey = editState.Parameters[editState.SelectedIndex].Key
			editState.CursorPos = len(editState.EditingKey)
		case request.QueryAddMode:
			// In add mode, tab switches between key and value fields
			if editState.CursorPos <= len(editState.EditingKey) {
				// Move to value field
				editState.CursorPos = len(editState.EditingKey) + len(editState.EditingValue)
			} else {
				// Move to key field
				editState.CursorPos = 0
			}
		}
		return m, nil

	case tea.KeyLeft:
		// Move cursor left
		if editState.CursorPos > 0 {
			editState.CursorPos--
		}
		return m, nil

	case tea.KeyRight:
		// Move cursor right
		maxPos := 0
		switch editState.Mode {
		case request.QueryEditKeyMode:
			maxPos = len(editState.EditingKey)
		case request.QueryEditValueMode:
			maxPos = len(editState.EditingValue)
		case request.QueryAddMode:
			maxPos = len(editState.EditingKey) + len(editState.EditingValue)
		}
		if editState.CursorPos < maxPos {
			editState.CursorPos++
		}
		return m, nil

	case tea.KeyUp:
		// Navigate to previous item while staying in edit mode
		if editState.SelectedIndex > 0 {
			// Save current edit first
			switch editState.Mode {
			case request.QueryEditKeyMode:
				if editState.SelectedIndex < len(editState.Parameters) {
					editState.Parameters[editState.SelectedIndex].Key = editState.EditingKey
					m.currentReq.SyncQueryToMap()
				}
			case request.QueryEditValueMode:
				if editState.SelectedIndex < len(editState.Parameters) {
					editState.Parameters[editState.SelectedIndex].Value = editState.EditingValue
					m.currentReq.SyncQueryToMap()
				}
			}
			
			// Move to previous item and load its data
			editState.SelectedIndex--
			switch editState.Mode {
			case request.QueryEditKeyMode:
				editState.EditingKey = editState.Parameters[editState.SelectedIndex].Key
				editState.CursorPos = len(editState.EditingKey)
			case request.QueryEditValueMode:
				editState.EditingValue = editState.Parameters[editState.SelectedIndex].Value
				editState.CursorPos = len(editState.EditingValue)
			}
		}
		return m, nil

	case tea.KeyDown:
		// Navigate to next item while staying in edit mode
		if editState.SelectedIndex < len(editState.Parameters)-1 {
			// Save current edit first
			switch editState.Mode {
			case request.QueryEditKeyMode:
				if editState.SelectedIndex < len(editState.Parameters) {
					editState.Parameters[editState.SelectedIndex].Key = editState.EditingKey
					m.currentReq.SyncQueryToMap()
				}
			case request.QueryEditValueMode:
				if editState.SelectedIndex < len(editState.Parameters) {
					editState.Parameters[editState.SelectedIndex].Value = editState.EditingValue
					m.currentReq.SyncQueryToMap()
				}
			}
			
			// Move to next item and load its data
			editState.SelectedIndex++
			switch editState.Mode {
			case request.QueryEditKeyMode:
				editState.EditingKey = editState.Parameters[editState.SelectedIndex].Key
				editState.CursorPos = len(editState.EditingKey)
			case request.QueryEditValueMode:
				editState.EditingValue = editState.Parameters[editState.SelectedIndex].Value
				editState.CursorPos = len(editState.EditingValue)
			}
		}
		return m, nil

	case tea.KeyBackspace:
		// Delete character before cursor
		switch editState.Mode {
		case request.QueryEditKeyMode:
			if editState.CursorPos > 0 && editState.CursorPos <= len(editState.EditingKey) {
				editState.EditingKey = editState.EditingKey[:editState.CursorPos-1] + editState.EditingKey[editState.CursorPos:]
				editState.CursorPos--
			}
		case request.QueryEditValueMode:
			if editState.CursorPos > 0 && editState.CursorPos <= len(editState.EditingValue) {
				editState.EditingValue = editState.EditingValue[:editState.CursorPos-1] + editState.EditingValue[editState.CursorPos:]
				editState.CursorPos--
			}
		case request.QueryAddMode:
			keyLen := len(editState.EditingKey)
			if editState.CursorPos > 0 && editState.CursorPos <= keyLen {
				// Editing key
				editState.EditingKey = editState.EditingKey[:editState.CursorPos-1] + editState.EditingKey[editState.CursorPos:]
				editState.CursorPos--
			} else if editState.CursorPos > keyLen {
				// Editing value
				valuePos := editState.CursorPos - keyLen
				if valuePos > 0 && valuePos <= len(editState.EditingValue) {
					editState.EditingValue = editState.EditingValue[:valuePos-1] + editState.EditingValue[valuePos:]
					editState.CursorPos--
				}
			}
		}
		return m, nil

	case tea.KeyRunes:
		// Insert character at cursor
		if len(msg.Runes) == 1 {
			char := string(msg.Runes[0])
			switch editState.Mode {
			case request.QueryEditKeyMode:
				editState.EditingKey = editState.EditingKey[:editState.CursorPos] + char + editState.EditingKey[editState.CursorPos:]
				editState.CursorPos++
			case request.QueryEditValueMode:
				editState.EditingValue = editState.EditingValue[:editState.CursorPos] + char + editState.EditingValue[editState.CursorPos:]
				editState.CursorPos++
			case request.QueryAddMode:
				keyLen := len(editState.EditingKey)
				if editState.CursorPos <= keyLen {
					// Editing key
					editState.EditingKey = editState.EditingKey[:editState.CursorPos] + char + editState.EditingKey[editState.CursorPos:]
					editState.CursorPos++
				} else {
					// Editing value
					valuePos := editState.CursorPos - keyLen
					editState.EditingValue = editState.EditingValue[:valuePos] + char + editState.EditingValue[valuePos:]
					editState.CursorPos++
				}
			}
		}
		return m, nil
	}

	return m, nil
}


func (h *RequestInputHandler) handleHeaderTextInput(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	if m.currentReq == nil || m.currentReq.HeaderEditState == nil {
		return m, nil
	}

	editState := m.currentReq.HeaderEditState

	switch msg.Type {
	case tea.KeyEsc:
		// Exit edit mode
		editState.Mode = request.HeaderViewMode
		editState.EditingKey = ""
		editState.EditingValue = ""
		editState.CursorPos = 0
		return m, nil

	case tea.KeyEnter:
		// Save the current edit
		switch editState.Mode {
		case request.HeaderEditKeyMode:
			// Save key edit
			if editState.SelectedIndex < len(editState.Parameters) {
				editState.Parameters[editState.SelectedIndex].Key = editState.EditingKey
				m.currentReq.SyncHeadersToMap()
			}
			editState.Mode = request.HeaderViewMode
			editState.EditingKey = ""
			editState.CursorPos = 0
		case request.HeaderEditValueMode:
			// Save value edit
			if editState.SelectedIndex < len(editState.Parameters) {
				editState.Parameters[editState.SelectedIndex].Value = editState.EditingValue
				m.currentReq.SyncHeadersToMap()
			}
			editState.Mode = request.HeaderViewMode
			editState.EditingValue = ""
			editState.CursorPos = 0
		case request.HeaderAddMode:
			// Add new header
			if editState.EditingKey != "" {
				newParam := request.HeaderParameter{
					Key:   editState.EditingKey,
					Value: editState.EditingValue,
				}
				editState.Parameters = append(editState.Parameters, newParam)
				m.currentReq.SyncHeadersToMap()
				editState.Mode = request.HeaderViewMode
				editState.EditingKey = ""
				editState.EditingValue = ""
				editState.CursorPos = 0
				editState.SelectedIndex = len(editState.Parameters) - 1
			}
		}
		return m, nil

	case tea.KeyTab:
		// Switch between key and value editing
		switch editState.Mode {
		case request.HeaderEditKeyMode:
			// Switch to editing value
			editState.Mode = request.HeaderEditValueMode
			editState.EditingValue = editState.Parameters[editState.SelectedIndex].Value
			editState.CursorPos = len(editState.EditingValue)
		case request.HeaderEditValueMode:
			// Switch to editing key  
			editState.Mode = request.HeaderEditKeyMode
			editState.EditingKey = editState.Parameters[editState.SelectedIndex].Key
			editState.CursorPos = len(editState.EditingKey)
		case request.HeaderAddMode:
			// In add mode, tab switches between key and value fields
			if editState.CursorPos <= len(editState.EditingKey) {
				// Move to value field
				editState.CursorPos = len(editState.EditingKey) + len(editState.EditingValue)
			} else {
				// Move to key field
				editState.CursorPos = 0
			}
		}
		return m, nil

	case tea.KeyLeft:
		// Move cursor left
		if editState.CursorPos > 0 {
			editState.CursorPos--
		}
		return m, nil

	case tea.KeyRight:
		// Move cursor right
		maxPos := 0
		switch editState.Mode {
		case request.HeaderEditKeyMode:
			maxPos = len(editState.EditingKey)
		case request.HeaderEditValueMode:
			maxPos = len(editState.EditingValue)
		case request.HeaderAddMode:
			maxPos = len(editState.EditingKey) + len(editState.EditingValue)
		}
		if editState.CursorPos < maxPos {
			editState.CursorPos++
		}
		return m, nil

	case tea.KeyUp:
		// Navigate to previous item while staying in edit mode
		if editState.SelectedIndex > 0 {
			// Save current edit first
			switch editState.Mode {
			case request.HeaderEditKeyMode:
				if editState.SelectedIndex < len(editState.Parameters) {
					editState.Parameters[editState.SelectedIndex].Key = editState.EditingKey
					m.currentReq.SyncHeadersToMap()
				}
			case request.HeaderEditValueMode:
				if editState.SelectedIndex < len(editState.Parameters) {
					editState.Parameters[editState.SelectedIndex].Value = editState.EditingValue
					m.currentReq.SyncHeadersToMap()
				}
			}
			
			// Move to previous item and load its data
			editState.SelectedIndex--
			switch editState.Mode {
			case request.HeaderEditKeyMode:
				editState.EditingKey = editState.Parameters[editState.SelectedIndex].Key
				editState.CursorPos = len(editState.EditingKey)
			case request.HeaderEditValueMode:
				editState.EditingValue = editState.Parameters[editState.SelectedIndex].Value
				editState.CursorPos = len(editState.EditingValue)
			}
		}
		return m, nil

	case tea.KeyDown:
		// Navigate to next item while staying in edit mode
		if editState.SelectedIndex < len(editState.Parameters)-1 {
			// Save current edit first
			switch editState.Mode {
			case request.HeaderEditKeyMode:
				if editState.SelectedIndex < len(editState.Parameters) {
					editState.Parameters[editState.SelectedIndex].Key = editState.EditingKey
					m.currentReq.SyncHeadersToMap()
				}
			case request.HeaderEditValueMode:
				if editState.SelectedIndex < len(editState.Parameters) {
					editState.Parameters[editState.SelectedIndex].Value = editState.EditingValue
					m.currentReq.SyncHeadersToMap()
				}
			}
			
			// Move to next item and load its data
			editState.SelectedIndex++
			switch editState.Mode {
			case request.HeaderEditKeyMode:
				editState.EditingKey = editState.Parameters[editState.SelectedIndex].Key
				editState.CursorPos = len(editState.EditingKey)
			case request.HeaderEditValueMode:
				editState.EditingValue = editState.Parameters[editState.SelectedIndex].Value
				editState.CursorPos = len(editState.EditingValue)
			}
		}
		return m, nil

	case tea.KeyBackspace:
		// Delete character before cursor
		switch editState.Mode {
		case request.HeaderEditKeyMode:
			if editState.CursorPos > 0 && editState.CursorPos <= len(editState.EditingKey) {
				editState.EditingKey = editState.EditingKey[:editState.CursorPos-1] + editState.EditingKey[editState.CursorPos:]
				editState.CursorPos--
			}
		case request.HeaderEditValueMode:
			if editState.CursorPos > 0 && editState.CursorPos <= len(editState.EditingValue) {
				editState.EditingValue = editState.EditingValue[:editState.CursorPos-1] + editState.EditingValue[editState.CursorPos:]
				editState.CursorPos--
			}
		case request.HeaderAddMode:
			keyLen := len(editState.EditingKey)
			if editState.CursorPos > 0 && editState.CursorPos <= keyLen {
				// Editing key
				editState.EditingKey = editState.EditingKey[:editState.CursorPos-1] + editState.EditingKey[editState.CursorPos:]
				editState.CursorPos--
			} else if editState.CursorPos > keyLen {
				// Editing value
				valuePos := editState.CursorPos - keyLen
				if valuePos > 0 && valuePos <= len(editState.EditingValue) {
					editState.EditingValue = editState.EditingValue[:valuePos-1] + editState.EditingValue[valuePos:]
					editState.CursorPos--
				}
			}
		}
		return m, nil

	case tea.KeyRunes:
		// Insert character at cursor
		if len(msg.Runes) == 1 {
			char := string(msg.Runes[0])
			switch editState.Mode {
			case request.HeaderEditKeyMode:
				editState.EditingKey = editState.EditingKey[:editState.CursorPos] + char + editState.EditingKey[editState.CursorPos:]
				editState.CursorPos++
			case request.HeaderEditValueMode:
				editState.EditingValue = editState.EditingValue[:editState.CursorPos] + char + editState.EditingValue[editState.CursorPos:]
				editState.CursorPos++
			case request.HeaderAddMode:
				keyLen := len(editState.EditingKey)
				if editState.CursorPos <= keyLen {
					// Editing key
					editState.EditingKey = editState.EditingKey[:editState.CursorPos] + char + editState.EditingKey[editState.CursorPos:]
					editState.CursorPos++
				} else {
					// Editing value
					valuePos := editState.CursorPos - keyLen
					editState.EditingValue = editState.EditingValue[:valuePos] + char + editState.EditingValue[valuePos:]
					editState.CursorPos++
				}
			}
		}
		return m, nil
	}

	return m, nil
}

func (h *RequestInputHandler) handleRequestSectionAction(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	if m.currentReq == nil {
		return m, nil
	}

	switch m.requestCursor {
	case request.QuerySection:
		return h.handleQuerySectionAction(m, msg)
	case request.BodySection:
		return h.handleBodySectionAction(m, msg)
	case request.HeadersSection:
		return h.handleHeadersSectionAction(m, msg)
	case request.AuthSection:
		return h.handleAuthSectionAction(m, msg)
	default:
		return m, nil
	}
}

func (h *RequestInputHandler) handleQuerySectionAction(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	// Initialize edit state if needed
	m.currentReq.InitializeQueryEditState()
	editState := m.currentReq.QueryEditState

	// Handle navigation and actions in view mode
	if editState.Mode == request.QueryViewMode {
		switch msg.Type {
		case tea.KeyUp:
			if editState.SelectedIndex > 0 {
				editState.SelectedIndex--
			}
			return m, nil
		case tea.KeyDown:
			if editState.SelectedIndex < len(editState.Parameters)-1 {
				editState.SelectedIndex++
			}
			return m, nil
		case tea.KeyEnter:
			// Enter edit mode for key
			if len(editState.Parameters) == 0 {
				// Create new parameter and enter add mode
				editState.Mode = request.QueryAddMode
				editState.EditingKey = ""
				editState.EditingValue = ""
				editState.CursorPos = 0
			} else if editState.SelectedIndex < len(editState.Parameters) {
				editState.Mode = request.QueryEditKeyMode
				editState.EditingKey = editState.Parameters[editState.SelectedIndex].Key
				editState.CursorPos = len(editState.EditingKey)
			}
			return m, nil
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "i":
				// Enter edit mode for key
				if len(editState.Parameters) == 0 {
					// Create new parameter and enter add mode
					editState.Mode = request.QueryAddMode
					editState.EditingKey = ""
					editState.EditingValue = ""
					editState.CursorPos = 0
				} else if editState.SelectedIndex < len(editState.Parameters) {
					editState.Mode = request.QueryEditKeyMode
					editState.EditingKey = editState.Parameters[editState.SelectedIndex].Key
					editState.CursorPos = len(editState.EditingKey)
				}
				return m, nil
			case "a":
				// Add new parameter
				editState.Mode = request.QueryAddMode
				editState.EditingKey = ""
				editState.EditingValue = ""
				editState.CursorPos = 0
				return m, nil
			case "d":
				// Delete parameter
				if editState.SelectedIndex < len(editState.Parameters) {
					editState.PendingDeletion = editState.SelectedIndex
				}
				return m, nil
			case "y":
				// Confirm deletion
				if editState.PendingDeletion >= 0 && editState.PendingDeletion < len(editState.Parameters) {
					// Remove parameter
					editState.Parameters = append(editState.Parameters[:editState.PendingDeletion], editState.Parameters[editState.PendingDeletion+1:]...)
					editState.PendingDeletion = -1
					if editState.SelectedIndex >= len(editState.Parameters) && len(editState.Parameters) > 0 {
						editState.SelectedIndex = len(editState.Parameters) - 1
					}
					m.currentReq.SyncQueryToMap()
				}
				return m, nil
			case "n":
				// Cancel deletion
				editState.PendingDeletion = -1
				return m, nil
			}
		}
	}

	return m, nil
}

func (h *RequestInputHandler) handleBodySectionAction(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	// Initialize edit state if needed
	m.currentReq.InitializeBodyEditState()
	editState := m.currentReq.BodyEditState

	// Handle navigation and actions in view mode
	if editState.Mode == request.BodyViewMode {
		switch msg.Type {
		case tea.KeyEnter:
			// Enter edit mode
			editState.Mode = request.BodyTextEditMode
			// Position cursor at end of content
			lines := strings.Split(editState.Content, "\n")
			if len(lines) > 0 {
				editState.CursorLine = len(lines) - 1
				editState.CursorCol = len(lines[len(lines)-1])
			}
			return m, nil
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "i":
				// Same as Enter - edit body
				editState.Mode = request.BodyTextEditMode
				lines := strings.Split(editState.Content, "\n")
				if len(lines) > 0 {
					editState.CursorLine = len(lines) - 1
					editState.CursorCol = len(lines[len(lines)-1])
				}
				return m, nil
			}
		}
	}

	return m, nil
}

func (h *RequestInputHandler) handleHeadersSectionAction(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	// Initialize edit state if needed
	m.currentReq.InitializeHeaderEditState()
	editState := m.currentReq.HeaderEditState

	// Handle navigation and actions in view mode
	if editState.Mode == request.HeaderViewMode {
		switch msg.Type {
		case tea.KeyUp:
			if editState.SelectedIndex > 0 {
				editState.SelectedIndex--
			}
			return m, nil
		case tea.KeyDown:
			if editState.SelectedIndex < len(editState.Parameters)-1 {
				editState.SelectedIndex++
			}
			return m, nil
		case tea.KeyEnter:
			// Enter edit mode for key
			if len(editState.Parameters) == 0 {
				// Create new header and enter add mode
				editState.Mode = request.HeaderAddMode
				editState.EditingKey = ""
				editState.EditingValue = ""
				editState.CursorPos = 0
			} else if editState.SelectedIndex < len(editState.Parameters) {
				editState.Mode = request.HeaderEditKeyMode
				editState.EditingKey = editState.Parameters[editState.SelectedIndex].Key
				editState.CursorPos = len(editState.EditingKey)
			}
			return m, nil
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "i":
				// Enter edit mode for key
				if len(editState.Parameters) == 0 {
					// Create new header and enter add mode
					editState.Mode = request.HeaderAddMode
					editState.EditingKey = ""
					editState.EditingValue = ""
					editState.CursorPos = 0
				} else if editState.SelectedIndex < len(editState.Parameters) {
					editState.Mode = request.HeaderEditKeyMode
					editState.EditingKey = editState.Parameters[editState.SelectedIndex].Key
					editState.CursorPos = len(editState.EditingKey)
				}
				return m, nil
			case "a":
				// Add new header
				editState.Mode = request.HeaderAddMode
				editState.EditingKey = ""
				editState.EditingValue = ""
				editState.CursorPos = 0
				return m, nil
			case "d":
				// Delete header
				if editState.SelectedIndex < len(editState.Parameters) {
					editState.PendingDeletion = editState.SelectedIndex
				}
				return m, nil
			case "y":
				// Confirm deletion
				if editState.PendingDeletion >= 0 && editState.PendingDeletion < len(editState.Parameters) {
					// Remove header
					editState.Parameters = append(editState.Parameters[:editState.PendingDeletion], editState.Parameters[editState.PendingDeletion+1:]...)
					editState.PendingDeletion = -1
					if editState.SelectedIndex >= len(editState.Parameters) && len(editState.Parameters) > 0 {
						editState.SelectedIndex = len(editState.Parameters) - 1
					}
					m.currentReq.SyncHeadersToMap()
				}
				return m, nil
			case "n":
				// Cancel deletion
				editState.PendingDeletion = -1
				return m, nil
			}
		}
	}

	return m, nil
}



func (h *RequestInputHandler) handleAuthTextInput(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	if m.currentReq == nil || m.currentReq.AuthEditState == nil {
		return m, nil
	}

	editState := m.currentReq.AuthEditState

	switch msg.Type {
	case tea.KeyEsc:
		// Exit edit mode
		editState.Mode = request.AuthViewMode
		editState.CursorPos = 0
		return m, nil

	case tea.KeyEnter:
		// Save the current edit and exit edit mode
		switch editState.Mode {
		case request.AuthTypeSelectMode:
			editState.Mode = request.AuthViewMode
			m.currentReq.SyncAuthToRequest()
		case request.AuthBearerEditMode:
			editState.Mode = request.AuthViewMode
			m.currentReq.SyncAuthToRequest()
		case request.AuthBasicUsernameEditMode:
			editState.Mode = request.AuthViewMode
			m.currentReq.SyncAuthToRequest()
		case request.AuthBasicPasswordEditMode:
			editState.Mode = request.AuthViewMode
			m.currentReq.SyncAuthToRequest()
		}
		editState.CursorPos = 0
		return m, nil

	case tea.KeyTab:
		// Switch between username and password in basic auth mode
		if editState.AuthType == "basic" {
			switch editState.Mode {
			case request.AuthBasicUsernameEditMode:
				editState.Mode = request.AuthBasicPasswordEditMode
				editState.CursorPos = len(editState.BasicPassword)
			case request.AuthBasicPasswordEditMode:
				editState.Mode = request.AuthBasicUsernameEditMode
				editState.CursorPos = len(editState.BasicUsername)
			}
		}
		return m, nil

	case tea.KeyUp, tea.KeyDown:
		// Handle type selection dropdown
		if editState.Mode == request.AuthTypeSelectMode {
			if msg.Type == tea.KeyUp {
				switch editState.AuthType {
				case "bearer":
					editState.AuthType = "none"
				case "basic":
					editState.AuthType = "bearer"
				}
			} else { // KeyDown
				switch editState.AuthType {
				case "none":
					editState.AuthType = "bearer"
				case "bearer":
					editState.AuthType = "basic"
				}
			}
		}
		return m, nil

	case tea.KeyLeft:
		// Move cursor left
		if editState.CursorPos > 0 {
			editState.CursorPos--
		}
		return m, nil

	case tea.KeyRight:
		// Move cursor right
		maxPos := 0
		switch editState.Mode {
		case request.AuthBearerEditMode:
			maxPos = len(editState.BearerToken)
		case request.AuthBasicUsernameEditMode:
			maxPos = len(editState.BasicUsername)
		case request.AuthBasicPasswordEditMode:
			maxPos = len(editState.BasicPassword)
		}
		if editState.CursorPos < maxPos {
			editState.CursorPos++
		}
		return m, nil

	case tea.KeyBackspace:
		// Delete character before cursor
		switch editState.Mode {
		case request.AuthBearerEditMode:
			if editState.CursorPos > 0 && editState.CursorPos <= len(editState.BearerToken) {
				editState.BearerToken = editState.BearerToken[:editState.CursorPos-1] + editState.BearerToken[editState.CursorPos:]
				editState.CursorPos--
			}
		case request.AuthBasicUsernameEditMode:
			if editState.CursorPos > 0 && editState.CursorPos <= len(editState.BasicUsername) {
				editState.BasicUsername = editState.BasicUsername[:editState.CursorPos-1] + editState.BasicUsername[editState.CursorPos:]
				editState.CursorPos--
			}
		case request.AuthBasicPasswordEditMode:
			if editState.CursorPos > 0 && editState.CursorPos <= len(editState.BasicPassword) {
				editState.BasicPassword = editState.BasicPassword[:editState.CursorPos-1] + editState.BasicPassword[editState.CursorPos:]
				editState.CursorPos--
			}
		}
		return m, nil

	case tea.KeyRunes:
		// Insert character at cursor
		if len(msg.Runes) == 1 {
			char := string(msg.Runes[0])
			switch editState.Mode {
			case request.AuthBearerEditMode:
				editState.BearerToken = editState.BearerToken[:editState.CursorPos] + char + editState.BearerToken[editState.CursorPos:]
				editState.CursorPos++
			case request.AuthBasicUsernameEditMode:
				editState.BasicUsername = editState.BasicUsername[:editState.CursorPos] + char + editState.BasicUsername[editState.CursorPos:]
				editState.CursorPos++
			case request.AuthBasicPasswordEditMode:
				editState.BasicPassword = editState.BasicPassword[:editState.CursorPos] + char + editState.BasicPassword[editState.CursorPos:]
				editState.CursorPos++
			}
		}
		return m, nil
	}

	return m, nil
}

func (h *RequestInputHandler) handleAuthSectionAction(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	// Initialize edit state if needed
	m.currentReq.InitializeAuthEditState()
	editState := m.currentReq.AuthEditState

	// Handle navigation and actions in view mode
	if editState.Mode == request.AuthViewMode {
		switch msg.Type {
		case tea.KeyUp:
			maxField := 0
			switch editState.AuthType {
			case "bearer":
				maxField = 1 // type, token
			case "basic":
				maxField = 2 // type, username, password
			}
			if editState.SelectedField > 0 {
				editState.SelectedField--
			} else {
				editState.SelectedField = maxField
			}
			return m, nil
		case tea.KeyDown:
			maxField := 0
			switch editState.AuthType {
			case "bearer":
				maxField = 1 // type, token
			case "basic":
				maxField = 2 // type, username, password
			}
			if editState.SelectedField < maxField {
				editState.SelectedField++
			} else {
				editState.SelectedField = 0
			}
			return m, nil
		case tea.KeyEnter:
			// Enter edit mode for selected field
			switch editState.SelectedField {
			case 0:
				// Edit auth type
				editState.Mode = request.AuthTypeSelectMode
			case 1:
				// Edit first field (token or username)
				if editState.AuthType == "bearer" {
					editState.Mode = request.AuthBearerEditMode
					editState.CursorPos = len(editState.BearerToken)
				} else if editState.AuthType == "basic" {
					editState.Mode = request.AuthBasicUsernameEditMode
					editState.CursorPos = len(editState.BasicUsername)
				}
			case 2:
				// Edit password (basic auth only)
				if editState.AuthType == "basic" {
					editState.Mode = request.AuthBasicPasswordEditMode
					editState.CursorPos = len(editState.BasicPassword)
				}
			}
			return m, nil
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "i":
				// Same as Enter - edit selected field
				switch editState.SelectedField {
				case 0:
					editState.Mode = request.AuthTypeSelectMode
				case 1:
					if editState.AuthType == "bearer" {
						editState.Mode = request.AuthBearerEditMode
						editState.CursorPos = len(editState.BearerToken)
					} else if editState.AuthType == "basic" {
						editState.Mode = request.AuthBasicUsernameEditMode
						editState.CursorPos = len(editState.BasicUsername)
					}
				case 2:
					if editState.AuthType == "basic" {
						editState.Mode = request.AuthBasicPasswordEditMode
						editState.CursorPos = len(editState.BasicPassword)
					}
				}
				return m, nil
			}
		}
	}

	return m, nil
}

// handleBodyTextInput handles text input for the body editor
func (h *RequestInputHandler) handleBodyTextInput(m *model, msg tea.KeyMsg) (*model, tea.Cmd) {
	if m.currentReq == nil || m.currentReq.BodyEditState == nil {
		return m, nil
	}

	editState := m.currentReq.BodyEditState

	switch msg.Type {
	case tea.KeyEsc:
		// Exit edit mode
		editState.Mode = request.BodyViewMode
		return m, nil

	case tea.KeyCtrlS:
		// Format JSON if body type is JSON
		if m.currentReq.Body.Type == "json" {
			content := strings.TrimSpace(editState.Content)
			if content != "" {
				var jsonData interface{}
				if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
					// JSON is valid, format it
					if formattedBytes, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
						editState.Content = string(formattedBytes)
						// Reset cursor to start after formatting
						editState.CursorLine = 0
						editState.CursorCol = 0
					}
				}
			}
		}
		
		// Save and exit edit mode
		m.currentReq.SyncBodyToRequest()
		editState.Mode = request.BodyViewMode
		return m, nil

	case tea.KeyEnter:
		// Insert new line
		content := editState.Content
		lines := strings.Split(content, "\n")
		
		if editState.CursorLine < len(lines) {
			line := lines[editState.CursorLine]
			newLines := make([]string, len(lines)+1)
			copy(newLines[:editState.CursorLine], lines[:editState.CursorLine])
			newLines[editState.CursorLine] = line[:editState.CursorCol]
			newLines[editState.CursorLine+1] = line[editState.CursorCol:]
			copy(newLines[editState.CursorLine+2:], lines[editState.CursorLine+1:])
			
			editState.Content = strings.Join(newLines, "\n")
			editState.CursorLine++
			editState.CursorCol = 0
		} else {
			// Append new line
			editState.Content += "\n"
			editState.CursorLine++
			editState.CursorCol = 0
		}
		
		// Validate JSON if body type is JSON
		if m.currentReq.Body.Type == "json" {
			m.currentReq.ValidateBodyJSON()
		}
		return m, nil

	case tea.KeyUp:
		// Move cursor up
		if editState.CursorLine > 0 {
			editState.CursorLine--
			lines := strings.Split(editState.Content, "\n")
			if editState.CursorLine < len(lines) {
				line := lines[editState.CursorLine]
				if editState.CursorCol > len(line) {
					editState.CursorCol = len(line)
				}
			}
		}
		return m, nil

	case tea.KeyDown:
		// Move cursor down
		lines := strings.Split(editState.Content, "\n")
		if editState.CursorLine < len(lines)-1 {
			editState.CursorLine++
			if editState.CursorLine < len(lines) {
				line := lines[editState.CursorLine]
				if editState.CursorCol > len(line) {
					editState.CursorCol = len(line)
				}
			}
		}
		return m, nil

	case tea.KeyLeft:
		// Move cursor left
		if editState.CursorCol > 0 {
			editState.CursorCol--
		} else if editState.CursorLine > 0 {
			// Move to end of previous line
			editState.CursorLine--
			lines := strings.Split(editState.Content, "\n")
			if editState.CursorLine < len(lines) {
				editState.CursorCol = len(lines[editState.CursorLine])
			}
		}
		return m, nil

	case tea.KeyRight:
		// Move cursor right
		lines := strings.Split(editState.Content, "\n")
		if editState.CursorLine < len(lines) {
			line := lines[editState.CursorLine]
			if editState.CursorCol < len(line) {
				editState.CursorCol++
			} else if editState.CursorLine < len(lines)-1 {
				// Move to beginning of next line
				editState.CursorLine++
				editState.CursorCol = 0
			}
		}
		return m, nil

	case tea.KeyHome:
		// Move to beginning of line
		editState.CursorCol = 0
		return m, nil

	case tea.KeyEnd:
		// Move to end of line
		lines := strings.Split(editState.Content, "\n")
		if editState.CursorLine < len(lines) {
			editState.CursorCol = len(lines[editState.CursorLine])
		}
		return m, nil

	case tea.KeyBackspace:
		// Delete character before cursor
		lines := strings.Split(editState.Content, "\n")
		if editState.CursorCol > 0 {
			// Delete character in current line
			if editState.CursorLine < len(lines) {
				line := lines[editState.CursorLine]
				newLine := line[:editState.CursorCol-1] + line[editState.CursorCol:]
				lines[editState.CursorLine] = newLine
				editState.Content = strings.Join(lines, "\n")
				editState.CursorCol--
			}
		} else if editState.CursorLine > 0 {
			// Join with previous line
			prevLine := lines[editState.CursorLine-1]
			currentLine := lines[editState.CursorLine]
			newLines := make([]string, len(lines)-1)
			copy(newLines[:editState.CursorLine-1], lines[:editState.CursorLine-1])
			newLines[editState.CursorLine-1] = prevLine + currentLine
			copy(newLines[editState.CursorLine:], lines[editState.CursorLine+1:])
			
			editState.Content = strings.Join(newLines, "\n")
			editState.CursorLine--
			editState.CursorCol = len(prevLine)
		}
		
		// Validate JSON if body type is JSON
		if m.currentReq.Body.Type == "json" {
			m.currentReq.ValidateBodyJSON()
		}
		return m, nil

	case tea.KeyDelete:
		// Delete character at cursor
		lines := strings.Split(editState.Content, "\n")
		if editState.CursorLine < len(lines) {
			line := lines[editState.CursorLine]
			if editState.CursorCol < len(line) {
				// Delete character in current line
				newLine := line[:editState.CursorCol] + line[editState.CursorCol+1:]
				lines[editState.CursorLine] = newLine
				editState.Content = strings.Join(lines, "\n")
			} else if editState.CursorLine < len(lines)-1 {
				// Join with next line
				nextLine := lines[editState.CursorLine+1]
				newLines := make([]string, len(lines)-1)
				copy(newLines[:editState.CursorLine], lines[:editState.CursorLine])
				newLines[editState.CursorLine] = line + nextLine
				copy(newLines[editState.CursorLine+1:], lines[editState.CursorLine+2:])
				
				editState.Content = strings.Join(newLines, "\n")
			}
		}
		
		// Validate JSON if body type is JSON
		if m.currentReq.Body.Type == "json" {
			m.currentReq.ValidateBodyJSON()
		}
		return m, nil

	case tea.KeyRunes:
		// Insert character at cursor
		if len(msg.Runes) == 1 {
			char := string(msg.Runes[0])
			lines := strings.Split(editState.Content, "\n")
			
			if editState.CursorLine < len(lines) {
				line := lines[editState.CursorLine]
				newLine := line[:editState.CursorCol] + char + line[editState.CursorCol:]
				lines[editState.CursorLine] = newLine
				editState.Content = strings.Join(lines, "\n")
				editState.CursorCol++
			} else {
				// Append to end
				editState.Content += char
				editState.CursorCol++
			}
			
			// Validate JSON if body type is JSON
			if m.currentReq.Body.Type == "json" {
				m.currentReq.ValidateBodyJSON()
			}
		}
		return m, nil
	}

	return m, nil
}

