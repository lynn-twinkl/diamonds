package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const dataFileName = "data.json"
const configDirName = "diamonds"

func getDataFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not get user config dir: %w", err)
	}

	appConfigDir := filepath.Join(configDir, configDirName)
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("could not create app config dir: %w", err)
	}

	return filepath.Join(appConfigDir, dataFileName), nil
}

func (m *model) saveProjects() {
	path, err := getDataFilePath()
	if err != nil {
		m.message = fmt.Sprintf("Error getting data path: %v", err)
		return
	}

	data, err := json.MarshalIndent(m.projects, "", "  ")
	if err != nil {
		m.message = fmt.Sprintf("Error saving data: %v", err)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		m.message = fmt.Sprintf("Error writing data: %v", err)
	}
}

func loadProjects() ([]Project, error) {
	path, err := getDataFilePath()
	if err != nil {
		return nil, fmt.Errorf("could not get data file path: %w", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Project{}, nil // No file, start fresh
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read data file: %w", err)
	}

	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, fmt.Errorf("could not parse data file: %w", err)
	}

	return projects, nil
}

// ViewState determines which view is currently active.
type ViewState int

const (
	ProjectListView ViewState = iota
	ColorListView
	AddProjectView
	AddColorView
	UrlListView
	AddUrlView
	ProjectMenuView
)

// --- LIST ITEM (Project) ---
type projectItem struct {
	name       string
	colorCount int
	urlCount   int
}

func (p projectItem) FilterValue() string { return p.name }
func (p projectItem) Title() string       { return p.name }
func (p projectItem) Description() string {
	colorStr := "colors"
	if p.colorCount == 1 {
		colorStr = "color"
	}
	urlStr := "URLs"
	if p.urlCount == 1 {
		urlStr = "URL"
	}
	return fmt.Sprintf("%d %s, %d %s", p.colorCount, colorStr, p.urlCount, urlStr)
}

// --- MODEL ---
type namedURL struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Project struct {
	Name   string     `json:"name"`
	Colors []string   `json:"colors"`
	Urls   []namedURL `json:"urls"`
}

type model struct {
	projectList     list.Model
	projects        []Project
	currentView     ViewState
	cursor          int
	selectedProject int
	inputBuffer     string // Used for single-line inputs
	urlNameBuffer   string // Used for the URL name in AddUrlView
	focusedField    int    // Used in AddUrlView to track focus
	message         string
}

// --- STYLING PARAMETERS ---
var (
	// Pipe: Adaptive purple for app name/header
	appNameColor = lipgloss.AdaptiveColor{Light: "#1E90FF", Dark: "#F6FFFE"}
	// Comment: Gray text for secondary info
	commentColor = lipgloss.Color("#757575")
	// Flag: Adaptive color for selected items
	selectionColor = lipgloss.AdaptiveColor{Light: "#0000CD", Dark: "#BAF3EB"}
	itemDescColor = lipgloss.AdaptiveColor{Light: "#5151D8", Dark: "#E9F8F5"}
	// ErrorHeader: Used for status messages
	messageColor   = lipgloss.Color("#F1F1F1")
	messageBgColor = lipgloss.Color("#FF5F87")
	// InlineCode: Pink on a dark/light background
	inlineCodeColor   = lipgloss.Color("#FF5F87")
	inlineCodeBgColor = lipgloss.AdaptiveColor{Light: "#ADD8E6", Dark: "#3A3A3A"}
	// Quote: Adaptive pink for interactive elements
	quoteColor = lipgloss.AdaptiveColor{Light: "#1E90FF", Dark: "#FF59C8"}
	// Normal: For regular text
	normalTextColor = lipgloss.AdaptiveColor{Light: "#1F2026", Dark: "#E5E5E5"}

	// Styles built from the color parameters
	// AppName + Pipe
	headerStyle = lipgloss.NewStyle().
			Foreground(appNameColor).
			Bold(true).
			MarginBottom(1)

	// Comment
	helpStyle = lipgloss.NewStyle().
			Foreground(commentColor)

	subtleStyle = lipgloss.NewStyle().
			Foreground(commentColor)

	// ErrorHeader
	messageStyle = lipgloss.NewStyle().
			Foreground(messageColor).
			Background(messageBgColor).
			Bold(true).
			Padding(0, 1)

	// InlineCode
	inlineCodeStyle = lipgloss.NewStyle().
			Foreground(inlineCodeColor).
			Background(inlineCodeBgColor).
			Padding(0, 1).
			Bold(true)

	// Flag
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(selectionColor)

	// Quote
	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(quoteColor).
			Padding(1, 2).
			Width(40)

	// Doc: General styling for the whole app
	docStyle = lipgloss.NewStyle().Padding(2,1).Foreground(normalTextColor)
)

func newCustomDelegate() list.DefaultDelegate {  
	// Create a new default delegate  
	d := list.NewDefaultDelegate()  
  
	// Change colors (using your selection color)  
	c := selectionColor  
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(c).BorderLeftForeground(c)  
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(itemDescColor).BorderLeftForeground(c) // Add BorderLeftForeground  
  
	// Set color for normal (unselected) items  
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(normalTextColor)  
	d.Styles.NormalDesc = d.Styles.NormalDesc.Foreground(commentColor)  
  
	return d  
}

// --- INITIALIZATION & UPDATE LOGIC ---

func initialModel() model {
	loadedProjects, err := loadProjects()
	if err != nil {
		fmt.Printf("Error loading projects: %v\n", err)
		os.Exit(1)
	}

	items := make([]list.Item, len(loadedProjects))
	for i, project := range loadedProjects {
		items[i] = projectItem{name: project.Name, colorCount: len(project.Colors), urlCount: len(project.Urls)}
	}

	delegate := newCustomDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "🪩 DIAMONDS "
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = headerStyle.Copy().MarginTop(0).PaddingTop(1)
	l.Styles.HelpStyle = helpStyle
	l.SetShowHelp(false)

	return model{
		projectList: l,
		projects:    loadedProjects,
		currentView: ProjectListView,
	}
}

func (m *model) updateProjectListItems() {
	items := make([]list.Item, len(m.projects))
	for i, project := range m.projects {
		items[i] = projectItem{name: project.Name, colorCount: len(project.Colors), urlCount: len(project.Urls)}
	}
	m.projectList.SetItems(items)
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		h, v := docStyle.GetHorizontalPadding(), docStyle.GetVerticalPadding()
		m.projectList.SetSize(msg.Width-h, msg.Height-v)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.currentView {
		case ProjectListView:
			return m.updateProjectList(msg)
		case ProjectMenuView:
			return m.updateProjectMenu(msg)
		case ColorListView:
			return m.updateColorList(msg)
		case UrlListView:
			return m.updateUrlList(msg)
		case AddProjectView:
			return m.updateAddProject(msg)
		case AddColorView:
			return m.updateAddColor(msg)
		case AddUrlView:
			return m.updateAddUrl(msg)
		}
	}
	return m, nil
}

func (m *model) updateProjectList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "enter":
		selectedItem, ok := m.projectList.SelectedItem().(projectItem)
		if ok {
			for i, p := range m.projects {
				if p.Name == selectedItem.name {
					m.selectedProject = i
					m.currentView = ProjectMenuView
					m.cursor = 0
					break
				}
			}
		}
		return m, nil
	case "n":
		m.currentView = AddProjectView
		m.inputBuffer = ""
		return m, nil
	}
	var cmd tea.Cmd
	m.projectList, cmd = m.projectList.Update(msg)
	return m, cmd
}

func (m *model) updateProjectMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.currentView = ProjectListView
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 1 {
			m.cursor++
		}
	case "enter":
		if m.cursor == 0 {
			m.currentView = ColorListView
		} else {
			m.currentView = UrlListView
		}
		m.cursor = 0
	}
	return m, nil
}

func (m *model) updateColorList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.currentView = ProjectMenuView
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.projects[m.selectedProject].Colors)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.projects[m.selectedProject].Colors) > 0 {
			color := m.projects[m.selectedProject].Colors[m.cursor]
			clipboard.WriteAll(color)
			m.message = fmt.Sprintf(" Copied %s to clipboard! ", color)
		}
	case "n":
		m.currentView = AddColorView
		m.inputBuffer = ""
	}
	return m, nil
}

func (m *model) updateUrlList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.currentView = ProjectMenuView
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.projects[m.selectedProject].Urls)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.projects[m.selectedProject].Urls) > 0 {
			url := m.projects[m.selectedProject].Urls[m.cursor].URL
			clipboard.WriteAll(url)
			m.message = fmt.Sprintf(" Copied %s to clipboard! ", url)
		}
	case "n":
		m.currentView = AddUrlView
		m.inputBuffer = ""
		m.urlNameBuffer = ""
		m.focusedField = 0
	}
	return m, nil
}

func (m *model) updateAddProject(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.currentView = ProjectListView
		m.inputBuffer = ""
	case "enter":
		if m.inputBuffer != "" {
			m.projects = append(m.projects, Project{Name: m.inputBuffer, Colors: []string{}, Urls: []namedURL{}})
			m.updateProjectListItems()
			m.saveProjects()
			m.currentView = ProjectListView
			m.inputBuffer = ""
		}
	case "backspace":
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
		}
	case " ":
		m.inputBuffer += " "
	default:
		if msg.Type == tea.KeyRunes {
			m.inputBuffer += string(msg.Runes)
		}
	}
	return m, nil
}

func (m *model) updateAddColor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.currentView = ColorListView
		m.inputBuffer = ""
	case "enter":
		if m.inputBuffer != "" && strings.HasPrefix(m.inputBuffer, "#") && (len(m.inputBuffer) == 7 || len(m.inputBuffer) == 4) {
			m.projects[m.selectedProject].Colors = append(m.projects[m.selectedProject].Colors, m.inputBuffer)
			m.updateProjectListItems()
			m.saveProjects()
			m.currentView = ColorListView
			m.cursor = len(m.projects[m.selectedProject].Colors) - 1
			m.inputBuffer = ""
		}
	case "backspace":
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
		}
	default:
		if msg.Type == tea.KeyRunes && len(m.inputBuffer) < 7 {
			m.inputBuffer += string(msg.Runes)
		}
	}
	return m, nil
}

func (m *model) updateAddUrl(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.currentView = UrlListView
		m.urlNameBuffer = ""
		m.inputBuffer = ""
		m.focusedField = 0
	case "enter":
		if m.focusedField == 0 {
			m.focusedField = 1
		} else {
			if m.urlNameBuffer != "" && m.inputBuffer != "" {
				m.projects[m.selectedProject].Urls = append(m.projects[m.selectedProject].Urls, namedURL{Name: m.urlNameBuffer, URL: m.inputBuffer})
				m.updateProjectListItems()
				m.saveProjects()
				m.currentView = UrlListView
				m.cursor = len(m.projects[m.selectedProject].Urls) - 1
				m.urlNameBuffer = ""
				m.inputBuffer = ""
				m.focusedField = 0
			}
		}
	case "backspace":
		if m.focusedField == 0 {
			if len(m.urlNameBuffer) > 0 {
				m.urlNameBuffer = m.urlNameBuffer[:len(m.urlNameBuffer)-1]
			}
		} else {
			if len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			}
		}
	case "tab":
		m.focusedField = (m.focusedField + 1) % 2
	case " ":
		if m.focusedField == 0 {
			m.urlNameBuffer += " "
		} else {
			m.inputBuffer += " "
		}
	default:
		if msg.Type == tea.KeyRunes {
			if m.focusedField == 0 {
				m.urlNameBuffer += string(msg.Runes)
			} else {
				m.inputBuffer += string(msg.Runes)
			}
		}
	}
	return m, nil
}

// --- VIEWS ---

func (m *model) View() string {
	var view string
	switch m.currentView {
	case ProjectListView:
		view = m.viewProjectList()
	case ProjectMenuView:
		view = m.viewProjectMenu()
	case ColorListView:
		view = m.viewColorList()
	case UrlListView:
		view = m.viewUrlList()
	case AddProjectView:
		view = m.viewAddProject()
	case AddColorView:
		view = m.viewAddColor()
	case AddUrlView:
		view = m.viewAddUrl()
	}
	return docStyle.Render(view)
}

func (m *model) viewProjectList() string {
	var b strings.Builder
	b.WriteString(m.projectList.View())
	help := horizontalHelp("↑/↓ navigate", "n new item", "q quit", "? more")
	b.WriteString("\n" + help)

	if m.message != "" {
		b.WriteString("\n" + messageStyle.Render(m.message))
		m.message = ""
	}
	return b.String()
}

func (m *model) viewProjectMenu() string {  
    project := m.projects[m.selectedProject]  
    var b strings.Builder  
  
    b.WriteString(headerStyle.Render(fmt.Sprintf("✨ %s", project.Name)) + "\n")  
  
    options := []string{"Colors", "URLs"}  
    for i, option := range options {  
        if m.cursor == i {  
            b.WriteString(selectedItemStyle.Render("> " + option) + "\n")  
        } else {  
            b.WriteString("  " + option + "\n")  // Ensure exactly 2 spaces  
        }  
    }  
  
    help := horizontalHelp("↑/↓ navigate", "enter select", "esc back", "q quit")  
    b.WriteString("\n" + help)  
  
    return b.String()  
}

func (m *model) viewColorList() string {
	project := m.projects[m.selectedProject]
	var b strings.Builder

	b.WriteString(headerStyle.Render(fmt.Sprintf("%s", project.Name)) + "\n")

	if len(project.Colors) == 0 {
		b.WriteString(subtleStyle.Render("No colors yet. Press 'n' to add one.") + "\n")
	} else {
		for i, color := range project.Colors {
			// The unused 'cursor' and 'style' variables have been removed.

			colorBlock := lipgloss.NewStyle().Background(lipgloss.Color(color)).Render("  ")
			hexCodeStyled := inlineCodeStyle.Render(color)
			line := fmt.Sprintf("%s %s", colorBlock, hexCodeStyled)

			if m.cursor == i {
				// Style for the cursor: colored but NOT bold
				cursorStyle := lipgloss.NewStyle().Foreground(selectionColor)
				styledCursor := cursorStyle.Render("> ")

				// Style for the line: uses the existing bold and colored style
				styledLine := selectedItemStyle.Render(line)

				b.WriteString(styledCursor + styledLine + "\n")
			} else {
				// For unselected lines, just add padding
				b.WriteString("  " + line + "\n")
			}
		}
	}

	help := horizontalHelp("↑/↓ navigate", "enter copy", "n new color", "esc back", "q quit")
	b.WriteString("\n" + help)

	if m.message != "" {
		b.WriteString("\n" + messageStyle.Render(m.message))
		m.message = ""
	}

	return b.String()
}

func (m *model) viewUrlList() string {
	project := m.projects[m.selectedProject]
	var b strings.Builder

	b.WriteString(headerStyle.Render(fmt.Sprintf("%s", project.Name)) + "\n")

	if len(project.Urls) == 0 {
		b.WriteString(subtleStyle.Render("No URLs yet. Press 'n' to add one.") + "\n")
	} else {
		for i, namedUrl := range project.Urls {
			if m.cursor == i {
				b.WriteString(selectedItemStyle.Render("> " + namedUrl.Name) + "\n")
			} else {
				b.WriteString("  " + namedUrl.Name + "\n")
			}
		}
	}

	help := horizontalHelp("↑/↓ navigate", "enter copy", "n new URL", "esc back", "q quit")
	b.WriteString("\n" + help)

	if m.message != "" {
		b.WriteString("\n" + messageStyle.Render(m.message))
		m.message = ""
	}

	return b.String()
}

func (m *model) viewAddProject() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Add New Project") + "\n")
	prompt := fmt.Sprintf("Project name: %s", m.inputBuffer)
	b.WriteString(inputStyle.Render(prompt) + "\n\n")
	b.WriteString(horizontalHelp("enter save", "esc cancel"))
	return b.String()
}

func (m *model) viewAddColor() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Add New Color") + "\n")
	prompt := fmt.Sprintf("HEX color: %s", m.inputBuffer)
	b.WriteString(inputStyle.Render(prompt) + "\n\n")
	b.WriteString(helpStyle.Render("Enter HEX (e.g., #FF5F87)") + "\n")
	b.WriteString(horizontalHelp("enter save", "esc cancel"))
	return b.String()
}

func (m *model) viewAddUrl() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Add New URL") + "\n")

	namePrompt := fmt.Sprintf("Name: %s", m.urlNameBuffer)
	urlPrompt := fmt.Sprintf("URL: %s", m.inputBuffer)

	if m.focusedField == 0 {
		b.WriteString(inputStyle.Render(namePrompt) + "\n")
		b.WriteString(subtleStyle.Render(urlPrompt) + "\n\n")
	} else {
		b.WriteString(subtleStyle.Render(namePrompt) + "\n")
		b.WriteString(inputStyle.Render(urlPrompt) + "\n\n")
	}

	b.WriteString(horizontalHelp("enter next/save", "tab switch fields", "esc cancel"))
	return b.String()
}

func horizontalHelp(keys ...string) string {
	return helpStyle.Render(strings.Join(keys, " • "))
}

func main() {
	m := initialModel()
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
