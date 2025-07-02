package main  
  
import (  
    "fmt"  
    "os"  
    "strings"  
      
    tea "github.com/charmbracelet/bubbletea"  
    "github.com/atotto/clipboard"  
)  
  
type ViewState int  
  
const (  
    ProjectListView ViewState = iota  
    ColorListView  
    AddProjectView  
    AddColorView  
)  
  
type Project struct {  
    name   string  
    colors []string  
}  
  
type model struct {  
    projects        []Project  
    currentView     ViewState  
    cursor          int  
    selectedProject int  
    inputBuffer     string  
    message         string  
}


func initialModel() model {
    return model{
        projects:    []Project{},
        currentView: ProjectListView,
        cursor:      0,
    }
}

func (m model) Init() tea.Cmd {
    return nil
}


func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch m.currentView {
        case ProjectListView:
            return m.updateProjectList(msg)
        case ColorListView:
            return m.updateColorList(msg)
        case AddProjectView:
            return m.updateAddProject(msg)
        case AddColorView:
            return m.updateAddColor(msg)
        }
    }
    return m, nil
}

func (m model) updateProjectList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit
    case "up", "k":
        if m.cursor > 0 {
            m.cursor--
        }
    case "down", "j":
        if m.cursor < len(m.projects)-1 {
            m.cursor++
        }
    case "enter":
        if len(m.projects) > 0 {
            m.selectedProject = m.cursor
            m.currentView = ColorListView
            m.cursor = 0
        }
    case "n":
        m.currentView = AddProjectView
        m.inputBuffer = ""
    }
    return m, nil
}

func (m model) updateColorList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit
    case "esc":
        m.currentView = ProjectListView
        m.cursor = m.selectedProject
    case "up", "k":
        if m.cursor > 0 {
            m.cursor--
        }
    case "down", "j":
        if m.cursor < len(m.projects[m.selectedProject].colors)-1 {
            m.cursor++
        }
    case "c":
        if len(m.projects[m.selectedProject].colors) > 0 {
            color := m.projects[m.selectedProject].colors[m.cursor]
            clipboard.WriteAll(color)
            m.message = fmt.Sprintf("Copied %s to clipboard!", color)
        }
    case "n":
        m.currentView = AddColorView
        m.inputBuffer = ""
    }
    return m, nil
}

func (m model) updateAddProject(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c":
        return m, tea.Quit
    case "esc":
        m.currentView = ProjectListView
        m.inputBuffer = ""
    case "enter":
        if m.inputBuffer != "" {
            m.projects = append(m.projects, Project{name: m.inputBuffer, colors: []string{}})
            m.currentView = ProjectListView
            m.inputBuffer = ""
        }
    case "backspace":
        if len(m.inputBuffer) > 0 {
            m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
        }
    default:
        m.inputBuffer += msg.String()
    }
    return m, nil
}

func (m model) updateAddColor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c":
        return m, tea.Quit
    case "esc":
        m.currentView = ColorListView
        m.inputBuffer = ""
    case "enter":
        if m.inputBuffer != "" && strings.HasPrefix(m.inputBuffer, "#") && len(m.inputBuffer) == 7 {
            m.projects[m.selectedProject].colors = append(m.projects[m.selectedProject].colors, m.inputBuffer)
            m.currentView = ColorListView
            m.inputBuffer = ""
        }
    case "backspace":
        if len(m.inputBuffer) > 0 {
            m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
        }
    default:
        if len(m.inputBuffer) < 7 {
            m.inputBuffer += msg.String()
        }
    }
    return m, nil
}

func (m model) View() string {
    switch m.currentView {
    case ProjectListView:
        return m.viewProjectList()
    case ColorListView:
        return m.viewColorList()
    case AddProjectView:
        return m.viewAddProject()
    case AddColorView:
        return m.viewAddColor()
    }
    return ""
}

func (m model) viewProjectList() string {
    s := "🎨 Color Manager - Projects\n\n"

    if len(m.projects) == 0 {
        s += "No projects yet. Press 'n' to create one.\n"
    } else {
        for i, project := range m.projects {
            cursor := " "
            if m.cursor == i {
                cursor = ">"
            }
            s += fmt.Sprintf("%s %s (%d colors)\n", cursor, project.name, len(project.colors))
        }
    }

    s += "\nControls:\n"
    s += "↑/↓ or k/j: Navigate\n"
    s += "Enter: Select project\n"
    s += "n: New project\n"
    s += "q: Quit\n"

    if m.message != "" {
        s += "\n" + m.message
        m.message = ""
    }

    return s
}

func (m model) viewColorList() string {
    project := m.projects[m.selectedProject]
    s := fmt.Sprintf("🎨 Project: %s\n\n", project.name)

    if len(project.colors) == 0 {
        s += "No colors yet. Press 'n' to add one.\n"
    } else {
        for i, color := range project.colors {
            cursor := " "
            if m.cursor == i {
                cursor = ">"
            }
            s += fmt.Sprintf("%s %s\n", cursor, color)
        }
    }

    s += "\nControls:\n"
    s += "↑/↓ or k/j: Navigate\n"
    s += "c: Copy color to clipboard\n"
    s += "n: New color\n"
    s += "Esc: Back to projects\n"
    s += "q: Quit\n"

    if m.message != "" {
        s += "\n" + m.message
    }

    return s
}

func (m model) viewAddProject() string {
    s := "🎨 Add New Project\n\n"
    s += fmt.Sprintf("Project name: %s\n\n", m.inputBuffer)
    s += "Enter project name and press Enter to save.\n"
    s += "Press Esc to cancel.\n"
    return s
}

func (m model) viewAddColor() string {
    s := "🎨 Add New Color\n\n"
    s += fmt.Sprintf("HEX color: %s\n\n", m.inputBuffer)
    s += "Enter HEX color (e.g., #FF5733) and press Enter to save.\n"
    s += "Press Esc to cancel.\n"
    return s
}



func main() {
    p := tea.NewProgram(initialModel())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %v", err)
        os.Exit(1)
    }
}
