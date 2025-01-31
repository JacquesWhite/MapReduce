package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

type model struct {
	localPath      string
	remotePath     string
	loadedEnv      bool
}

func initialModel() model {
	// Projects list
	projectsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	projectsList.Title = "Select GCP Project"
	projectsList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Padding(0, 1)
	projectsList.SetShowHelp(true)

	// Instance groups list
	instanceGroupsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	instanceGroupsList.Title = "Select Instance Group"
	instanceGroupsList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Padding(0, 1)
	instanceGroupsList.SetShowHelp(true)

	menuList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	menuList.Title = "Menu"
	menuList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Padding(0, 1)
	menuList.SetShowHelp(true)

	fetchingSpinner := spinner.New()
	fetchingSpinner.Spinner = spinner.Dot
	fetchingSpinner.Spinner.FPS = 10
	fetchingSpinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		projects:       projectsList,
		instanceGroups: instanceGroupsList,
		menu:           menuList,
		instancePicker: huh.NewForm(),
		spinner:        fetchingSpinner,
		output:         "",
		currentStep:    0,
		err:            nil,
		loadedEnv:      false,
	}
}

func (m model) Init() tea.Cmd {
	return loadEnvFileCmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			switch m.currentStep {
			case 0:
				if i, ok := m.projects.SelectedItem().(projectItem); ok {
					return m, projectSelected(i.projectId)
				}
			case 1:
				if i, ok := m.instanceGroups.SelectedItem().(instanceGroupItem); ok {
					return m, instanceGroupSelected(i.name, i.zone)
				}
			case 9:
				if m.loadedEnv {
					m.currentStep = 10
					return m, showMenuCmd
				}
			case 10:
				if i, ok := m.menu.SelectedItem().(menuItem); ok {
					switch i.title {
					case "Upload Folder":
						return m, uploadFolderCmd
					case "Run command":
						return m, runCommandCmd
					case "Quit":
						return m, tea.Quit
					}
				}
			case 11:
				if m.instancePicker.State == huh.StateCompleted {
					if m.instancePicker.State == huh.StateCompleted {
						fmt.Print(m.instancePicker.GetString("instance"))
					// fmt.Print(m.instancePicker.GetString("instance"))
					// fmt.Print(m.instancePicker.GetInt("level"))
					m.output = lipgloss.NewStyle().
						Width(100).
						BorderStyle(lipgloss.NormalBorder()).
						BorderForeground(lipgloss.Color("63")).
						Render(m.menu.View()) + "\n"
					m.currentStep = 11
					return m, uploadFileCmd
				}
				// case 12:
				// 	instancePicker, _ := m.instancePicker.Update(msg)
				// 	m.output = instancePicker.View()
				// 	m.currentStep = 12
				// 	cmds = append(cmds, m.instancePicker.NextGroup())
				// 	if m.instancePicker.State == huh.StateCompleted {
				// 		// fmt.Fprint(m.instancePicker.NextGroup())
				// 		// instancePicker.Update(msg)
				// 		// m.instancePicker.
				// 		// m.currentStep = 10
				// 		// return m, showMenuCmd
			}

		}

	case tea.WindowSizeMsg:
		m.projects.SetSize(msg.Width-2, 15)
		m.instanceGroups.SetSize(msg.Width-2, 15)
		m.menu.SetSize(msg.Width-2, msg.Height/2)

	case errMsg:
		m.err = msg
		return m, nil

	case loadedEnvFileMsg:
		if env.projectId == "" {
			m.output += "Getting list of projects from cloud projects list...\n"
			cmd = getProjectsCmd
		} else if env.zone == "" || env.instanceGroupName == "" {
			m.currentStep = 1
			m.output += "Getting list of instance groups from cloud compute instance-groups list...\n"
			cmd = getInstanceGroupsCmd
		} else if env.username == "" {
			m.currentStep = 2
			m.output += "Setting username to $USERNAME...\n"
			cmd = setUsernameCmd
		} else if env.ssh_key == "" {
			m.currentStep = 3
			cmd = setSSHKeyCmd
			m.output += "This step can take a while... output with ssh_keys is logged in ssh_key_setup.log\n"
		} else if len(instances) == 0 {
			m.currentStep = 4
			cmd = startFetchingInstancesCmd
		} else {
			m.loadedEnv = true
			m.currentStep = 9
			m.output += "Loaded environment file:\n"
			m.output += fmt.Sprintf("GCP_PROJECT_ID: %s\n", env.projectId)
			m.output += fmt.Sprintf("GCP_ZONE: %s\n", env.zone)
			m.output += fmt.Sprintf("GCP_INSTANCE_GROUP_NAME: %s\n", env.instanceGroupName)
			m.output += fmt.Sprintf("SSH_USERNAME: %s\n", env.username)
			m.output += fmt.Sprintf("SSH_KEY_PATH: %s\n", env.ssh_key)
		}
		cmds = append(cmds, cmd)

	case projectsMsg:
		var projectItems []list.Item
		for _, p := range msg.projects {
			projectItems = append(projectItems, projectItem{projectId: p})
		}
		m.projects.SetItems(projectItems)
		m.output += lipgloss.NewStyle().
			Width(100).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Render(m.projects.View()) + "\n"

	case instanceGroupsMsg:
		var instanceGroupItems []list.Item
		for _, ig := range msg.instanceGroups {
			instanceGroupItems = append(instanceGroupItems, instanceGroupItem{name: ig.name, zone: ig.zone})
		}
		m.instanceGroups.SetItems(instanceGroupItems)
		m.output += lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Width(100).
			Render(m.instanceGroups.View()) + "\n"

	case projectSelectedMsg:
		env.projectId = msg.projectId
		updateEnvFile("GCP_PROJECT_ID", env.projectId)
		m.output += fmt.Sprintf("Project selected: %s\n", env.projectId)
		m.currentStep = 1
		cmds = append(cmds, loadEnvFileCmd)

	case instanceGroupSelectedMsg:
		env.zone = msg.zone
		env.instanceGroupName = msg.name
		updateEnvFile("GCP_ZONE", env.zone)
		updateEnvFile("GCP_INSTANCE_GROUP_NAME", env.instanceGroupName)
		m.output += fmt.Sprintf("Instance group selected: %s (zone: %s)\n", env.instanceGroupName, env.zone)
		m.currentStep = 2
		cmds = append(cmds, loadEnvFileCmd)

	case usernameSetMsg:
		m.currentStep = 3
		cmds = append(cmds, loadEnvFileCmd)

	case startFetchingInstancesMsg:
		m.spinner.Tick()
		m.output += fmt.Sprintf("%s Fetching instance list...\n", m.spinner.View())
		cmds = append(cmds, fetchInstancesCmd)

	case fetchedInstancesMsg:
		m.currentStep = 4
		m.output += fmt.Sprintf("Fetched %d instances\n", len(instances))
		var instanceIPs []string
		for _, instance := range instances {
			instanceIPs = append(instanceIPs, instance.ExternalIP)
		}
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exPath := filepath.Dir(ex)
		m.instancePicker = huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Key("instances").
					Options(huh.NewOptions(instanceIPs...)...).
					Title("Select target instances"),

				huh.NewInput().
					Title("Specify local folder path.").
					Prompt(exPath+"/").
					Validate(checkLocalPath).
					Value(&m.localPath),
				huh.NewInput().
					Title("Specify remote folder path.").
					Prompt("~/").
					Value(&m.remotePath),
			))
		cmds = append(cmds, loadEnvFileCmd)

	case sshKeySetMsg:
		m.output += "SSH Key set up successfully!\n"
		m.loadedEnv = true
		cmds = append(cmds, loadEnvFileCmd)

	case showMenuMsg:
		var menuItems []list.Item
		menuItems = append(menuItems, menuItem{title: "Upload Folder", description: "Upload a folder to specific instances"})
		menuItems = append(menuItems, menuItem{title: "Run command", description: "Run a command on specific instances"})
		menuItems = append(menuItems, menuItem{title: "Quit", description: "Quit the program"})
		m.menu.SetItems(menuItems)
		m.output = lipgloss.NewStyle().
			Width(100).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Render(m.menu.View()) + "\n"

	case uploadFolderMsg:
		m.currentStep = 11
		cmds = append(cmds, m.instancePicker.Init())
		m.output = m.instancePicker.View()

	case runCommandMsg:
		m.currentStep = 12
		cmds = append(cmds, m.instancePicker.Init())
		m.output = m.instancePicker.View()
	}

	switch m.currentStep {
	case 0:
		if len(m.projects.Items()) == 0 {
			m.projects, cmd = m.projects.Update(msg)
			cmds = append(cmds, cmd)
		}
	case 1:
		if len(m.instanceGroups.Items()) == 0 {
			m.instanceGroups, cmd = m.instanceGroups.Update(msg)
			cmds = append(cmds, cmd)
		}
	case 3:
		m.spinner.Tick()
		m.spinner, cmd = m.spinner.Update(msg)
		output := m.output
		withoutLastLine := strings.Split(output, "\n")
		withoutLastLine = withoutLastLine[:len(withoutLastLine)-1]
		withoutLastLine = append(withoutLastLine, fmt.Sprintf("%s Fetching instance list...\n", m.spinner.View()))
		m.output = strings.Join(withoutLastLine, "\n")
		cmds = append(cmds, cmd)
	case 10:
		m.menu, cmd = m.menu.Update(msg)
		m.output = lipgloss.NewStyle().
			Width(100).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Render(m.menu.View()) + "\n"
		cmds = append(cmds, cmd)
	case 11:
		instancePicker, cmd := m.instancePicker.Update(msg)
		if f, ok := instancePicker.(*huh.Form); ok {
			m.instancePicker = f
			m.output = instancePicker.View()
		}
		cmds = append(cmds, cmd)
	case 12:

		// 	return m, nil
	}
	// }

	return m, tea.Batch(cmds...)
}

type projectsMsg struct {
	projects []string
}

func getProjectsCmd() tea.Msg {
	cmd := exec.Command("gcloud", "projects", "list", "--format=value(projectId)")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return errMsg(fmt.Errorf("Failed to execute gcloud command: %v. Install gcloud sdk and authenticate with `gcloud init`", err))
	}
	projectIDs := strings.Split(strings.TrimSpace(out.String()), "\n")
	return projectsMsg{projects: projectIDs}
}

type projectSelectedMsg struct {
	projectId string
}

func projectSelected(project string) tea.Cmd {
	return func() tea.Msg {
		return projectSelectedMsg{projectId: project}
	}
}

type instanceGroupsMsg struct {
	instanceGroups []instanceGroupItem
}

func getInstanceGroupsCmd() tea.Msg {
	cmd := exec.Command("gcloud", "compute", "instance-groups", "list", "--format=csv(name,zone)")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return errMsg(fmt.Errorf("Failed to execute gcloud command: %v. Install gcloud sdk and authenticate with `gcloud init`", err))
	}
	instanceGroupLines := strings.Split(strings.TrimSpace(out.String()), "\n")[1:]

	var instanceGroups []instanceGroupItem
	for _, line := range instanceGroupLines {
		parts := strings.Split(line, ",")
		locationParts := strings.Split(parts[1], "/")
		zone := locationParts[len(locationParts)-1]
		instanceGroups = append(instanceGroups, instanceGroupItem{name: parts[0], zone: zone})
	}
	return instanceGroupsMsg{instanceGroups: instanceGroups}
}

type instanceGroupSelectedMsg struct {
	name string
	zone string
}

func instanceGroupSelected(name, zone string) tea.Cmd {
	return func() tea.Msg {
		return instanceGroupSelectedMsg{name: name, zone: zone}
	}
}

type usernameSetMsg struct {
	username string
}

func setUsernameCmd() tea.Msg {
	if env.username == "" {
		// Get username from $USERNAME
		env.username = os.Getenv("USER")
		if env.username == "" {
			return errMsg(fmt.Errorf("Error: $USERNAME environment variable not set.\n"))
		}
		fmt.Println("Setting username to: %s (from $USERNAME)\n", env.username)
		updateEnvFile("SSH_USERNAME", env.username)
	}
	return usernameSetMsg{username: env.username}
}

type sshKeySetMsg struct{}

func setSSHKeyCmd() tea.Msg {

	if env.ssh_key == "" {

		if instances == nil {
			return startFetchingInstancesMsg{}
		}
		for _, instance := range instances {
			cmd := exec.Command("gcloud", "compute", "ssh", "--zone", env.zone, instance.InstanceId, "--project", env.projectId, "--command", "echo \"command\"")
			output, err := cmd.CombinedOutput()
			f, err := os.OpenFile("ssh_key_setup.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				panic(err)
			}
			defer f.Close()

			if _, err = f.Write(output); err != nil {
				panic(err)
			}
			if err != nil {
				return errMsg(fmt.Errorf("Failed to execute gcloud command: %v. Install gcloud sdk and authenticate with `gcloud init`", err))
			}
		}
		env.ssh_key = "/Users/" + os.Getenv("USER") + "/.ssh/google_compute_engine"
		updateEnvFile("SSH_KEY_PATH", env.ssh_key)
	}
	return loadedEnvFileMsg{}
}

type startFetchingInstancesMsg struct{}

func startFetchingInstancesCmd() tea.Msg {
	return startFetchingInstancesMsg{}
}

type fetchedInstancesMsg struct{}

func fetchInstancesCmd() tea.Msg {
	instancesRes, err := getInstances(context.Background(), env.projectId, env.zone, env.instanceGroupName)
	if err != nil {
		return errMsg(err)
	}
	instances = instancesRes
	return fetchedInstancesMsg{}
}

func updateEnvFile(key, value string) {
	fileContent, err := os.ReadFile(".env")
	if err != nil {
		fmt.Println("failed to read .env file: ", err)
	}
	newContent := strings.Replace(string(fileContent), key+"=", fmt.Sprintf("%s=%s", key, value), 1)
	err = os.WriteFile(".env", []byte(newContent), 0644)
	if err != nil {
		fmt.Println("failed to write to .env file: ", err)
	}
}

type showMenuMsg struct{}

func showMenuCmd() tea.Msg {
	return showMenuMsg{}
}

type uploadFolderMsg struct{}

func uploadFolderCmd() tea.Msg {
	return uploadFolderMsg{}
}

type runCommandMsg struct{}

func runCommandCmd() tea.Msg {
	return func() tea.Msg {
		return runCommandMsg{}
	}
}

func checkLocalPath(localPath string) error {
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("Local path does not exist: %s", localPath)
	}
	return nil
}

