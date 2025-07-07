package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
	Name string
	URL  string
}

type Project struct {
	name   string
	colors []string
	urls   []namedURL
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
	// Colors mapped from the style guide
	// Pipe: Adaptive purple for app name/header
	appNameColor = lipgloss.AdaptiveColor{Light: "#8470FF", Dark: "#745CFF"}
	// Comment: Gray text for secondary info
	commentColor = lipgloss.Color("#757575")
	// Flag: Adaptive green/cyan for selected items
	selectionColor = lipgloss.AdaptiveColor{Light: "#F780E2", Dark: "#F780E2"}
	// ErrorHeader: Used for status messages
	messageColor   = lipgloss.Color("#F1F1F1")
	messageBgColor = lipgloss.Color("#FF5F87")
	// InlineCode: Pink on a dark/light background
	inlineCodeColor   = lipgloss.Color("#FF5F87")
	inlineCodeBgColor = lipgloss.AdaptiveColor{Light: "#F2F2F2", Dark: "#3A3A3A"}
	// Quote: Adaptive pink for interactive elements
	quoteColor = lipgloss.AdaptiveColor{Light: "#FF71D0", Dark: "#FF78D2"}

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
				Foreground(selectionColor).
				Bold(true)

	// Quote
	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(quoteColor).
			Padding(1, 2).
			Width(40)
)

func newCustomDelegate() list.DefaultDelegate {
	// Create a new default delegate
	d := list.NewDefaultDelegate()

	// Change colors (using your selection color)
	c := selectionColor
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(c).BorderLeftForeground(c)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.Copy() // reuse the title style here

	return d
}

// --- INITIALIZATION & UPDATE LOGIC ---

func initialModel() model {
	initialProjects := []Project{
		{name: "Website Redesign", colors: []string{"#FF5733", "#33FF57", "#3357FF"}, urls: []namedURL{
			{Name: "Figma Mockups", URL: "https://www.figma.com/file/123"},
			{Name: "Design Specs", URL: "https://www.notion.so/design-specs"},
		}},
		{name: "Mobile App UI", colors: []string{"#C70039", "#900C3F"}, urls: []namedURL{
			{Name: "Figma Mockups", URL: "https://www.figma.com/file/456"},
		}},
		{name: "Branding Guide", colors: []string{"#F9E79F", "#5DADE2", "#1ABC9C", "#F1C40F"}, urls: []namedURL{}},
	}

	items := make([]list.Item, len(initialProjects))
	for i, project := range initialProjects {
		items[i] = projectItem{name: project.name, colorCount: len(project.colors), urlCount: len(project.urls)}
	}

	delegate := newCustomDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "🪩 Diamonds"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = headerStyle
	l.Styles.HelpStyle = helpStyle

	return model{
		projectList: l,
		projects:    initialProjects,
		currentView: ProjectListView,
	}
}

func (m *model) updateProjectListItems() {
	items := make([]list.Item, len(m.projects))
	for i, project := range m.projects {
		items[i] = projectItem{name: project.name, colorCount: len(project.colors), urlCount: len(project.urls)}
	}
	m.projectList.SetItems(items)
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.projectList.SetSize(msg.Width, msg.Height)
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
				if p.name == selectedItem.name {
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
		if m.cursor < len(m.projects[m.selectedProject].colors)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.projects[m.selectedProject].colors) > 0 {
			color := m.projects[m.selectedProject].colors[m.cursor]
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
		if m.cursor < len(m.projects[m.selectedProject].urls)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.projects[m.selectedProject].urls) > 0 {
			url := m.projects[m.selectedProject].urls[m.cursor].URL
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
			m.projects = append(m.projects, Project{name: m.inputBuffer, colors: []string{}, urls: []namedURL{}})
			m.updateProjectListItems()
			m.currentView = ProjectListView
			m.inputBuffer = ""
		}
	case "backspace":
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
		}
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
			m.projects[m.selectedProject].colors = append(m.projects[m.selectedProject].colors, m.inputBuffer)
			m.updateProjectListItems()
			m.currentView = ColorListView
			m.cursor = len(m.projects[m.selectedProject].colors) - 1
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
				m.projects[m.selectedProject].urls = append(m.projects[m.selectedProject].urls, namedURL{Name: m.urlNameBuffer, URL: m.inputBuffer})
				m.updateProjectListItems()
				m.currentView = UrlListView
				m.cursor = len(m.projects[m.selectedProject].urls) - 1
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
	switch m.currentView {
	case ProjectListView:
		return m.viewProjectList()
	case ProjectMenuView:
		return m.viewProjectMenu()
	case ColorListView:
		return m.viewColorList()
	case UrlListView:
		return m.viewUrlList()
	case AddProjectView:
		return m.viewAddProject()
	case AddColorView:
		return m.viewAddColor()
	case AddUrlView:
		return m.viewAddUrl()
	}
	return ""
}

func (m *model) viewProjectList() string {
	var b strings.Builder
	b.WriteString(m.projectList.View())

	if m.message != "" {
		b.WriteString("\n" + messageStyle.Render(m.message))
		m.message = ""
	}
	return b.String()
}

func (m *model) viewProjectMenu() string {
	project := m.projects[m.selectedProject]
	var b strings.Builder

	b.WriteString(headerStyle.Render(fmt.Sprintf("Project: %s", project.name)) + "\n")

	options := []string{"Colors", "URLs"}
	for i, option := range options {
		if m.cursor == i {
			b.WriteString(selectedItemStyle.Render("> " + option) + "\n")
		} else {
			b.WriteString("  " + option + "\n")
		}
	}

	help := "\n↑/↓: Navigate\n" + "Enter: Select\n" + "Esc: Back\n" + "q: Quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m *model) viewColorList() string {
	project := m.projects[m.selectedProject]
	var b strings.Builder

	b.WriteString(headerStyle.Render(fmt.Sprintf("Project: %s", project.name)) + "\n")

	if len(project.colors) == 0 {
		b.WriteString(subtleStyle.Render("No colors yet. Press 'n' to add one.") + "\n")
	} else {
		for i, color := range project.colors {
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

	help := "\n↑/↓: Navigate\n" + "Enter: Copy color\n" + "n: New color\n" + "Esc: Back\n" + "q: Quit"
	b.WriteString(helpStyle.Render(help))

	if m.message != "" {
		b.WriteString("\n" + messageStyle.Render(m.message))
		m.message = ""
	}

	return b.String()
}

func (m *model) viewUrlList() string {
	project := m.projects[m.selectedProject]
	var b strings.Builder

	b.WriteString(headerStyle.Render(fmt.Sprintf("Project: %s", project.name)) + "\n")

	if len(project.urls) == 0 {
		b.WriteString(subtleStyle.Render("No URLs yet. Press 'n' to add one.") + "\n")
	} else {
		for i, namedUrl := range project.urls {
			if m.cursor == i {
				b.WriteString(selectedItemStyle.Render("> " + namedUrl.Name) + "\n")
			} else {
				b.WriteString("  " + namedUrl.Name + "\n")
			}
		}
	}

	help := "\n↑/↓: Navigate\n" + "Enter: Copy URL\n" + "n: New URL\n" + "Esc: Back\n" + "q: Quit"
	b.WriteString(helpStyle.Render(help))

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
	b.WriteString(helpStyle.Render("Enter: Save\nEsc: Cancel"))
	return b.String()
}

func (m *model) viewAddColor() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Add New Color") + "\n")
	prompt := fmt.Sprintf("HEX color: %s", m.inputBuffer)
	b.WriteString(inputStyle.Render(prompt) + "\n\n")
	b.WriteString(helpStyle.Render("Enter HEX (e.g., #FF5F87)\nEnter: Save\nEsc: Cancel"))
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

	b.WriteString(helpStyle.Render("Enter: Next/Save\nTab: Switch Fields\nEsc: Cancel"))
	return b.String()
}

func main() {
	m := initialModel()
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
