package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// --- Styling ---
var (
	docStyle           = lipgloss.NewStyle().Margin(1, 2)
	titleStyle         = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Padding(0, 1)
	statusMessageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Italic(true)
	errorStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// --- Data Model ---
type portItem struct {
	port     uint32
	pid      int32
	procName string
}

func (i portItem) Title() string       { return fmt.Sprintf("Port %d", i.port) }
func (i portItem) Description() string { return fmt.Sprintf("PID: %d • %s", i.pid, i.procName) }
func (i portItem) FilterValue() string { return strconv.Itoa(int(i.port)) + i.procName }

type model struct {
	list    list.Model
	loading bool
	spinner spinner.Model
	message string
}

// --- Logic: Fetching Ports ---
func getActivePorts() []list.Item {
	conns, _ := net.Connections("tcp")
	seen := make(map[uint32]bool)
	var items []list.Item

	for _, c := range conns {
		if c.Status == "LISTEN" && !seen[c.Laddr.Port] {
			seen[c.Laddr.Port] = true
			name := "Unknown"
			if p, err := process.NewProcess(c.Pid); err == nil {
				name, _ = p.Name()
			}
			items = append(items, portItem{port: c.Laddr.Port, pid: c.Pid, procName: name})
		}
	}
	// Sort by port number
	sort.Slice(items, func(i, j int) bool {
		return items[i].(portItem).port < items[j].(portItem).port
	})
	return items
}

// --- Bubble Tea Core ---
func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r": // Refresh
			m.list.SetItems(getActivePorts())
			m.message = "List refreshed"
		case "x", "delete", "k": // Kill
			selected := m.list.SelectedItem().(portItem)
			p, err := process.NewProcess(selected.pid)
			if err == nil {
				err = p.Kill()
				if err != nil {
					m.message = errorStyle.Render(fmt.Sprintf("Failed to kill %d", selected.pid))
				} else {
					m.message = statusMessageStyle.Render(fmt.Sprintf("Successfully killed %s (Port %d)", selected.procName, selected.port))
					m.list.SetItems(getActivePorts()) // Auto-refresh
				}
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.list.View(),
			"\n"+m.message,
		),
	)
}

func main() {
	items := getActivePorts()

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Port Inspector & Executioner"
	// l.AdditionalFullHelpKeys = func() []list.KeyBinding {
	// 	return []list.KeyBinding{} // Add custom help here if needed
	// }

	m := model{list: l}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
