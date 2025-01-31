package gcp

import (
	"errors"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var menuStyle = lipgloss.NewStyle().Margin(1, 2)

type projectItem struct {
	GceProject
}

func (i projectItem) Title() string       { return string(i.ProjectId) }
func (i projectItem) Description() string { return string(i.ProjectURL) }
func (i projectItem) FilterValue() string { return string(i.ProjectId) }

func Item(projectURL ProjectURL) projectItem {
	return projectItem{GceProject{
		ProjectURL: projectURL,
		ProjectId:  getProjectIdFromProjectURL(projectURL),
	}}
}

type ProjectPickerModel struct {
	projects      list.Model
	pickedProject *GceProject
}

func (m ProjectPickerModel) Init() tea.Cmd {
	return nil
}

func New() ProjectPickerModel {
	projectsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	projectsList.Title = "Select a project"
	projectsList.SetShowFilter(false)
	projectsList.SetShowStatusBar(false)
	projectsList.SetFilteringEnabled(false)

	return ProjectPickerModel{
		projects:      projectsList,
		pickedProject: nil,
	}
}

type ProjecPickerPickedMsg struct{}

type ProjecPickerErrorMsg struct {
	err error
}

// Cmd's: ProjecPickerPickedMsg, ProjecPickerErrorMsg
func (m ProjectPickerModel) Update(msg tea.Msg) (ProjectPickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if i, ok := m.projects.SelectedItem().(projectItem); ok {
				m.pickedProject = &i.GceProject
				return m, Cmd(ProjecPickerPickedMsg{})
			}
		}
	case FetchProjectURLsSuccessMsg:
		var items []list.Item
		for _, projectURL := range msg.projectURLS {
			items = append(items, Item(projectURL))
		}
		m.projects.SetItems(items)
	case tea.WindowSizeMsg:
		h, v := menuStyle.GetFrameSize()
		m.projects.SetSize(msg.Width-h, msg.Height-v)
	}

	if len(m.projects.Items()) == 0 {
		return m, Cmd(ProjecPickerErrorMsg{err: errors.New("No projects found")})
	}

	var cmd tea.Cmd
	m.projects, cmd = m.projects.Update(msg)
	return m, cmd
}

func (m ProjectPickerModel) View() string {
	return menuStyle.Render(m.projects.View())
}

// Helper function to return a tea.Msg from function that returns tea.Cmd
func Cmd(msg tea.Msg) func() tea.Msg {
	return func() tea.Msg {
		return msg
	}
}
