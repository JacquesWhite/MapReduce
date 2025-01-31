package main

import (
	"MapReduce/deployment/menu"

	tea "github.com/charmbracelet/bubbletea"
)

type mainModel struct {
	envLoader envLoader.Model
	menu      menu.Model
	// envLoader envLoader.Model
	// uploader  uploader.Model
	// sshRunner sshRunner.Model

	// env       Env
	gceInstances []GceInstance
}

func (m mainModel) Init() tea.Cmd {
	return nil
}

type menuDelegate struct{}

func (m menuDelegate) RunCommandCmd() tea.Cmd {
	return tea.Quit
}

func (m menuDelegate) UploadFolderCmd() tea.Cmd {
	return tea.Quit
}

func (m menuDelegate) QuitCmd() tea.Cmd {
	return tea.Quit
}

func MainModel() mainModel {

	menu := menu.New(menuDelegate{})
	// envLoader := envLoader.NewModel()
	// uploader := uploader.NewModel()
	// sshRunner := sshRunner.NewModel()

	return mainModel{
		menu: menu,
		// envLoader: envLoader,
		// uploader:  uploader,
		// sshRunner: sshRunner,
	}
}
