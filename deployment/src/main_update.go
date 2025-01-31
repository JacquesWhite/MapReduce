package main

import tea "github.com/charmbracelet/bubbletea"

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	}

	// Pass messages to the menu model.
	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)

	// Return the updated model without a command to the Bubble Tea runtime for processing.
	return m, cmd
}
