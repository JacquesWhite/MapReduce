package gcp

import (
	"bytes"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type FetchProjectURLsSuccessMsg struct{ projectURLS []ProjectURL }
type FetchProjectURLsErrorMsg struct{ err error }

// Cmd's: FetchProjectURLsSuccessMsg, FetchProjectURLsErrorMsg
func FetchProjectURLs() tea.Cmd {
	cmd := exec.Command("gcloud", "projects", "list", "--format=value(projectId)")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return Cmd(FetchProjectURLsErrorMsg{err: err})
	}
	projectURLs := strings.Split(strings.TrimSpace(out.String()), "\n")
	var ProjectURLs []ProjectURL
	for _, projectURL := range projectURLs {
		ProjectURLs = append(ProjectURLs, ProjectURL(projectURL))
	}
	return Cmd(FetchProjectURLsSuccessMsg{projectURLS: ProjectURLs})
}
