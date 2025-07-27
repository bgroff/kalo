package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
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


type model struct {
	width           int
	height          int
	activePanel     panel
	collections     []panels.CollectionItem
	selectedReq     int
	bruRequests     []*panels.BruRequest
	currentReq      *panels.BruRequest
	requestCursor   panels.RequestSection
	response        string
	statusCode      int
	httpClient      *HTTPClient
	lastResponse    *panels.HTTPResponse
	isLoading       bool
	responseViewport viewport.Model
	headersViewport viewport.Model
	responseCursor  panels.ResponseSection
	commandPalette  *CommandPalette
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

func initialModel() model {
	responseVP := viewport.New(30, 10)
	responseVP.SetContent(`{
  "message": "Select a request to see response"
}`)

	headersVP := viewport.New(30, 5)
	headersVP.SetContent("No headers available")

	m := model{
		activePanel:      collectionsPanel,
		selectedReq:      0,
		requestCursor:    panels.URLSection,
		responseCursor:   panels.ResponseHeadersSection,
		statusCode:       200,
		httpClient:       NewHTTPClient(),
		responseViewport: responseVP,
		headersViewport:  headersVP,
		commandPalette:   NewCommandPalette(),
		response: `{
  "message": "Select a request to see response"
}`,
	}
	m.loadBruFiles()
	return m
}

func (m *model) loadBruFiles() {
	m.collections = []panels.CollectionItem{
		{Name: "📁 Examples", Type: "folder", IsFolder: true},
	}
	m.bruRequests = []*panels.BruRequest{}

	examplesDir := "examples"
	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		return
	}

	filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".bru") {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			parser := NewBruParser(file)
			request, err := parser.Parse()
			if err != nil {
				return err
			}

			m.bruRequests = append(m.bruRequests, request)
			
			methodColor := getMethodColor(request.HTTP.Method)
			displayName := fmt.Sprintf("  %s %s", methodColor, request.Meta.Name)
			
			m.collections = append(m.collections, panels.CollectionItem{
				Name:     displayName,
				Type:     "request",
				FilePath: path,
				IsFolder: false,
			})
		}
		return nil
	})

	if len(m.bruRequests) > 0 {
		m.currentReq = m.bruRequests[0]
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
		// Handle command palette input first
		if m.commandPalette.IsVisible() {
			switch msg.String() {
			case "esc":
				m.commandPalette.Hide()
				return m, nil
			case "enter":
				if cmd := m.commandPalette.GetSelectedCommand(); cmd != nil {
					m.commandPalette.Hide()
					return m, m.executeCommand(cmd.Action)
				}
				return m, nil
			case "up":
				m.commandPalette.MoveCursor(-1)
				return m, nil
			case "down":
				m.commandPalette.MoveCursor(1)
				return m, nil
			case "backspace":
				input := m.commandPalette.GetInput()
				if len(input) > 0 {
					m.commandPalette.SetInput(input[:len(input)-1])
				}
				return m, nil
			default:
				// Handle regular character input
				if len(msg.String()) == 1 && msg.String()[0] >= 32 && msg.String()[0] <= 126 {
					m.commandPalette.SetInput(m.commandPalette.GetInput() + msg.String())
				}
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "ctrl+p":
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
				m.selectedReq--
				m.updateCurrentRequest()
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
				m.selectedReq++
				m.updateCurrentRequest()
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
				m.updateCurrentRequest()
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
			if m.activePanel == responsePanel {
				if m.responseCursor == panels.ResponseHeadersSection {
					m.headersViewport.HalfViewDown()
				} else {
					m.responseViewport.HalfViewDown()
				}
			}
		case "pgup":
			if m.activePanel == responsePanel {
				if m.responseCursor == panels.ResponseHeadersSection {
					m.headersViewport.HalfViewUp()
				} else {
					m.responseViewport.HalfViewUp()
				}
			}
		case "home":
			if m.activePanel == responsePanel {
				if m.responseCursor == panels.ResponseHeadersSection {
					m.headersViewport.GotoTop()
				} else {
					m.responseViewport.GotoTop()
				}
			}
		case "end":
			if m.activePanel == responsePanel {
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
			m.response = m.httpClient.FormatResponseForDisplay(msg.response)
			m.statusCode = msg.response.StatusCode
			m.responseViewport.SetContent(m.response)
			m.responseViewport.GotoTop()
			
			// Format headers for display
			var headersContent strings.Builder
			if len(msg.response.Headers) > 0 {
				for key, value := range msg.response.Headers {
					headersContent.WriteString(fmt.Sprintf("%s: %s\n", key, value))
				}
			} else {
				headersContent.WriteString("No headers received")
			}
			m.headersViewport.SetContent(headersContent.String())
			m.headersViewport.GotoTop()
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
	if item.IsFolder {
		return
	}

	requestIndex := m.selectedReq - 1
	if requestIndex >= 0 && requestIndex < len(m.bruRequests) {
		m.currentReq = m.bruRequests[requestIndex]
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

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	sidebarWidth := m.width / 3
	mainWidth := m.width - sidebarWidth - 2
	
	// Account for header (1 line) + footer (1 line) + some padding
	availableHeight := m.height - 4

	sidebar := panels.RenderCollections(sidebarWidth, availableHeight, m.collections, m.selectedReq, m.activePanel == collectionsPanel, focusedStyle, blurredStyle, titleStyle)
	main := m.renderMainArea(mainWidth, availableHeight)

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		main,
	)

	header := titleStyle.Width(m.width - 2).Render("Kalo - Bruno API Client")
	
	var footerText string
	if m.isLoading {
		footerText = "Loading... • Tab: Switch panels • q: Quit"
	} else if m.activePanel == responsePanel {
		footerText = "Tab: Switch panels • ←→: Switch Headers/Body • ↑↓: Scroll • Space/PgDn/PgUp/Home/End • s: Send • q: Quit"
	} else {
		footerText = "Tab: Switch panels • ↑↓: Navigate • Enter/Space/s: Execute • q: Quit"
	}
	footer := headerStyle.Width(m.width - 2).Render(footerText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
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
	response := panels.RenderResponse(width, responseHeight, m.activePanel == responsePanel, m.isLoading, m.lastResponse, m.statusCode, m.responseCursor, &m.headersViewport, &m.responseViewport, focusedStyle, blurredStyle, titleStyle, cursorStyle, sectionStyle, statusOkStyle)

	return lipgloss.JoinVertical(lipgloss.Left, request, response)
}



func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}