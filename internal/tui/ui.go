package tui

import (
	"fmt"
	"github.com/Aryagorjipour/smartctl/internal/systemd"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Start       key.Binding
	Stop        key.Binding
	Enable      key.Binding
	Disable     key.Binding
	Restart     key.Binding
	Search      key.Binding
	Quit        key.Binding
	Escape      key.Binding
	Filter      key.Binding
	ClearFilter key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Start, k.Stop, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Start, k.Stop, k.Enable, k.Disable, k.Restart},
		{k.Search, k.Filter, k.ClearFilter},
		{k.Quit, k.Escape},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "go up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "go down"),
	),
	Start: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start service"),
	),
	Stop: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "stop service"),
	),
	Enable: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "enable service"),
	),
	Disable: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "disable service"),
	),
	Restart: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restart service"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	),
	ClearFilter: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear filter"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "exit"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

type Model struct {
	list        list.Model
	services    []systemd.Service
	filterState string
	searching   bool
	searchInput textinput.Model
	help        help.Model
	err         error
}

type serviceItem struct {
	service systemd.Service
}

func (i serviceItem) Title() string {
	return i.service.Name
}

func (i serviceItem) Description() string {
	status := "stopped"
	if i.service.Status == "running" {
		status = "running"
	}

	enabled := "disabled"
	if i.service.Enabled {
		enabled = "enabled"
	}

	return fmt.Sprintf("[%s | %s] %s", status, enabled, i.service.Description)
}

func (i serviceItem) FilterValue() string {
	return i.service.Name + " " + i.service.Description
}

func NewProgram() *tea.Program {
	return tea.NewProgram(initialModel(), tea.WithAltScreen())
}

func initialModel() Model {

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#DDDDDD")).
		Background(lipgloss.Color("#7D56F4"))

	m := Model{
		list:        list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		help:        help.New(),
		searchInput: textinput.New(),
	}

	m.list.Title = "SmartCTL: manage Systemd"
	m.list.SetStatusBarItemName("service", "services")

	m.list.Styles.Title = m.list.Styles.Title.Margin(0, 0, 1, 2)

	m.list.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.Start,
			keys.Stop,
			keys.Enable,
			keys.Disable,
			keys.Restart,
			keys.Search,
			keys.Filter,
			keys.ClearFilter,
		}
	}

	m.searchInput.Placeholder = "search ..."
	m.searchInput.CharLimit = 100
	m.searchInput.Width = 30

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadServices,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil
	case tea.KeyMsg:
		if m.searching {
			switch {
			case key.Matches(msg, keys.Escape):
				m.searching = false
				m.searchInput.Reset()
				return m, nil
			case msg.Type == tea.KeyEnter:
				m.searching = false
				searchTerm := m.searchInput.Value()
				m.searchInput.Reset()
				return m, filterServices(searchTerm, m.services)
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Search):
			m.searching = true
			return m, textinput.Blink
		case key.Matches(msg, keys.Filter):
			return m, tea.Sequence(
				filterRunning,
				func() tea.Msg {
					m.filterState = "running"
					return nil
				},
			)
		case key.Matches(msg, keys.ClearFilter):
			m.filterState = ""
			return m, loadServices
		case key.Matches(msg, keys.Start):
			if i, ok := m.list.SelectedItem().(serviceItem); ok {
				cmd := startService(i.service.Name)
				return m, tea.Sequence(cmd, loadServices)
			}
		case key.Matches(msg, keys.Stop):
			if i, ok := m.list.SelectedItem().(serviceItem); ok {
				cmd := stopService(i.service.Name)
				return m, tea.Sequence(cmd, loadServices)
			}
		case key.Matches(msg, keys.Enable):
			if i, ok := m.list.SelectedItem().(serviceItem); ok {
				cmd := enableService(i.service.Name)
				return m, tea.Sequence(cmd, loadServices)
			}
		case key.Matches(msg, keys.Disable):
			if i, ok := m.list.SelectedItem().(serviceItem); ok {
				cmd := disableService(i.service.Name)
				return m, tea.Sequence(cmd, loadServices)
			}
		case key.Matches(msg, keys.Restart):
			if i, ok := m.list.SelectedItem().(serviceItem); ok {
				cmd := restartService(i.service.Name)
				return m, tea.Sequence(cmd, loadServices)
			}
		}
	case servicesMsg:
		m.services = msg
		items := make([]list.Item, len(msg))
		for i, svc := range msg {
			items[i] = serviceItem{service: svc}
		}
		m.list.SetItems(items)
	case errorMsg:
		m.err = msg
	}

	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("ERROR: %v\n\n for exit the program press 'q'\n", m.err)
	}

	var s strings.Builder

	if m.searching {
		s.WriteString("\n  ")
		s.WriteString(m.searchInput.View())
		s.WriteString("\n\n  press 'Enter' for searching and 'Esc' for cancel\n")
		return s.String()
	}

	filterIndicator := ""
	if m.filterState != "" {
		filterIndicator = fmt.Sprintf(" [Filter: %s]", m.filterState)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render(m.list.Title + filterIndicator)

	s.WriteString(title)
	s.WriteString("\n")
	s.WriteString(m.list.View())

	helpView := m.help.View(keys)
	s.WriteString("\n")
	s.WriteString(helpView)

	return s.String()
}

type servicesMsg []systemd.Service
type errorMsg error

func loadServices() tea.Msg {
	services, err := systemd.ListServices()
	if err != nil {
		return errorMsg(err)
	}
	return servicesMsg(services)
}

func filterServices(term string, services []systemd.Service) tea.Cmd {
	return func() tea.Msg {
		var filtered []systemd.Service
		for _, s := range services {
			if strings.Contains(strings.ToLower(s.Name), strings.ToLower(term)) ||
				strings.Contains(strings.ToLower(s.Description), strings.ToLower(term)) {
				filtered = append(filtered, s)
			}
		}
		return servicesMsg(filtered)
	}
}

func filterRunning() tea.Msg {
	services, err := systemd.ListServices()
	if err != nil {
		return errorMsg(err)
	}

	var filtered []systemd.Service
	for _, s := range services {
		if s.Status == "running" {
			filtered = append(filtered, s)
		}
	}
	return servicesMsg(filtered)
}

func startService(name string) tea.Cmd {
	return func() tea.Msg {
		err := systemd.StartService(name)
		if err != nil {
			return errorMsg(fmt.Errorf("error on starting the service %s: %v", name, err))
		}
		return nil
	}
}

func stopService(name string) tea.Cmd {
	return func() tea.Msg {
		err := systemd.StopService(name)
		if err != nil {
			return errorMsg(fmt.Errorf("error on stop the service %s: %v", name, err))
		}
		return nil
	}
}

func enableService(name string) tea.Cmd {
	return func() tea.Msg {
		err := systemd.EnableService(name)
		if err != nil {
			return errorMsg(fmt.Errorf("error on enabeling the service %s: %v", name, err))
		}
		return nil
	}
}

func disableService(name string) tea.Cmd {
	return func() tea.Msg {
		err := systemd.DisableService(name)
		if err != nil {
			return errorMsg(fmt.Errorf("error on disabling the service %s: %v", name, err))
		}
		return nil
	}
}

func restartService(name string) tea.Cmd {
	return func() tea.Msg {
		err := systemd.RestartService(name)
		if err != nil {
			return errorMsg(fmt.Errorf("error on restarting the service %s: %v", name, err))
		}
		return nil
	}
}
