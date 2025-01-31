package envLoader

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	envLoaderStyle        = lipgloss.NewStyle().Margin(1, 2)
	currentOperationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	doneStyle             = lipgloss.NewStyle().Margin(1, 2)
	checkMark             = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
	warningMark           = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).SetString("⚠")
	greenStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

type installedPkgMsg string

func (m Model) View() string {
	if m.done {
		return doneStyle.Render(printEnv(*m.env))
	}
	spinner := m.spinner.View() + " "
	progress := m.progress.View()
	cellsAvailable := m.width - lipgloss.Width(spinner)
	info := lipgloss.NewStyle().MaxWidth(cellsAvailable).Render(m.currentOperation)
	return spinner + info + "\n" + progress
}

func printEnv(env Env) string {
	envStr := fmt.Sprintf("%s %s", checkMark, greenStyle.SetString("Loaded Env:\n"))
	return envStr
}
