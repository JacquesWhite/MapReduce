package envLoader

import (
	"os"

	"MapReduce/deployment/gcp"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

type Model struct {
	env              *Env
	operations       []string
	currentOperation string
	index            int
	width            int
	height           int
	spinner          spinner.Model
	progress         progress.Model
	delegate         EnvLoaderDelegate
	done             bool
}

type EnvLoaderDelegate interface {
	projectPickerModel() gcp.ProjectPickerModel
}

func New(envLoaderDelegate EnvLoaderDelegate) Model {
	progress := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)
	spinner := spinner.New()
	spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	return Model{
		operations: []string{},
		index:      0,
		spinner:    spinner,
		progress:   progress,
		done:       false,
	}
}

func (m Model) Init() tea.Cmd {
	m.currentOperation = "Loading environment file from .env"
	return tea.Batch(LoadEnvFileFromDotEnvCmd(), m.spinner.Tick)
}

type loadEnvFileFromDotEnvError struct {
	err error
}
type loadEnvFileFromDotEnvSuccess struct {
	env Env
}

func LoadEnvFileFromDotEnvCmd() tea.Cmd {
	err := godotenv.Overload(".env")
	if err != nil {
		Cmd(loadEnvFileFromDotEnvError{})
	}
	var env Env
	env.projectId = os.Getenv("GCP_PROJECT_ID")
	env.zone = os.Getenv("GCP_ZONE")
	env.instanceGroupName = os.Getenv("GCP_INSTANCE_GROUP_NAME")
	env.username = os.Getenv("SSH_USERNAME")
	env.ssh_key = os.Getenv("SSH_KEY_PATH")
	return Cmd(loadEnvFileFromDotEnvSuccess{env: env})
}

type loadEnvTemplateFileErrorMsg struct {
	err error
}
type loadEnvTemplateFileSuccessMsg struct {
	envTemplateFile []byte
}

func LoadEnvTemplateFileCmd() tea.Cmd {
	envTemplateFile, err := os.ReadFile(".env.template")
	if err != nil {
		return Cmd(loadEnvTemplateFileErrorMsg{err: err})
	}
	return Cmd(loadEnvTemplateFileSuccessMsg{envTemplateFile: envTemplateFile})
}

type createEnvFileFromTemplateErrorMsg struct {
	err error
}
type createEnvFileFromTemplateSuccessMsg struct{}

func CreateEnvFileFromTemplateCmd(envTemplateFile []byte) tea.Cmd {
	err := os.WriteFile(".env", envTemplateFile, 0644)
	if err != nil {
		return Cmd(createEnvFileFromTemplateErrorMsg{err: err})
	}
	return Cmd(createEnvFileFromTemplateSuccessMsg{})
}

func LoadProjectIdCmd() tea.Cmd {
	return gcp.FetchProjectURLs()
}

// Helper function to return a tea.Msg from function that returns tea.Cmd
func Cmd(msg tea.Msg) func() tea.Msg {
	return func() tea.Msg {
		return msg
	}
}
