package progress

import (
	tea "github.com/charmbracelet/bubbletea"
)

type IProgram interface {
	Run() (tea.Model, error)
	Quit()
	Add(key, fileName string)
	Update(key string, progress float64, status Status, msgFormat ...any)
}
