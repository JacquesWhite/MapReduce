package menu

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var menuStyle = lipgloss.NewStyle().Margin(1, 2)

type Model struct {
	items    list.Model
	delegate MenuDelegate
}

type MenuDelegate interface {
	UploadFolderCmd() tea.Cmd
	RunCommandCmd() tea.Cmd
	QuitCmd() tea.Cmd
}

func (m Model) Init() tea.Cmd {
	return nil
}

func New(menuDelegate MenuDelegate) Model {
	itemsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	items := []list.Item{}
	items = append(items, Item("Upload Folder", "Upload a folder to specific instances"))
	items = append(items, Item("Run command", "Run a command on specific instances"))
	items = append(items, Item("Quit", "Quit the program"))
	itemsList.SetItems(items)
	itemsList.Title = "Main Menu"
	itemsList.SetShowFilter(false)
	itemsList.SetShowStatusBar(false)
	itemsList.SetFilteringEnabled(false)

	return Model{
		items:    itemsList,
		delegate: menuDelegate,
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if i, ok := m.items.SelectedItem().(MenuItem); ok {
				switch i.title {
				case "Upload Folder":
					return m, m.delegate.UploadFolderCmd()
				case "Run command":
					return m, m.delegate.RunCommandCmd()
				case "Quit":
					return m, m.delegate.QuitCmd()
				}
			}
		}

	case tea.WindowSizeMsg:
		h, v := menuStyle.GetFrameSize()
		m.items.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.items, cmd = m.items.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return menuStyle.Render(m.items.View())
}
