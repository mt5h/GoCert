package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go-cert/checker"
	"io"
	"os"
	"strings"
	"time"
)

const listHeight = 14
const useHighPerformanceRenderer = false

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()

	textStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprint(w, fn(str))
}

type State int

const (
	Listing State = iota
	Checking
	Showing
)

type model struct {
	list     list.Model
	spinner  spinner.Model
	choice   string
	quitting bool
	state    State
	sub      chan string // where we'll receive activity notifications
	viewport viewport.Model
	ready    bool
}

func (m model) Init() tea.Cmd {
	return tea.Batch(waitForActivity(m.sub))
}

func checkServer(server string, sub chan string) {
	res := checker.GetJsonCert(server, 30*time.Second)
	time.Sleep(2 * time.Second)
	sub <- res
}

// A message used to indicate that activity has occurred. In the real world (for
// example, chat) this would contain actual data.
type responseMsg string

// A command that waits for the activity on a channel.
func waitForActivity(sub chan string) tea.Cmd {
	return func() tea.Msg {
		return responseMsg(<-sub)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.ready = true
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
		// Handle keyboard and mouse events in the viewport
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch m.state {
	case Listing:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch keypress := msg.String(); keypress {
			case "enter", "right", "l":
				currentItem, ok := m.list.SelectedItem().(item)
				if ok {
					m.choice = string(currentItem)
					m.state = Checking
					go checkServer(string(currentItem), m.sub)
					spinnerCmd := m.spinner.Tick
					return m, spinnerCmd
				}
			case "down", "j":
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
			case "up", "k":
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	case Showing:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch keypress := msg.String(); keypress {
			case "left", "h":
				m.state = Listing
			case "down", "j":
				m.viewport.ViewDown()
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			case "up", "k":
				m.viewport.ViewUp()
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	case Checking:
		switch msg := msg.(type) {
		case responseMsg:
			m.resetSpinner()
			m.state = Showing
			m.viewport.SetContent(string(msg))
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
			cmds = append(cmds, waitForActivity(m.sub))
		case spinner.TickMsg:
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		case tea.KeyMsg:
			switch keypress := msg.String(); keypress {
			}
		}
	}

	return m, tea.Batch(cmds...)

}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m model) headerView() string {
	title := titleStyle.Render("Certificate details")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}
func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m model) View() string {
	s := ""
	switch m.state {
	case Listing:
		s = "\n" + m.list.View()
	case Checking:
		s = fmt.Sprintf("\n  %s%s\n\n", m.spinner.View(), textStyle("Checking..."))
	case Showing:
		if !m.ready {
			s = "\n  Initializing..."
		} else {
			s = fmt.Sprintf("\n%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
		}
	}
	return s
}

func (m *model) resetSpinner() {
	m.spinner = spinner.New()
	m.spinner.Style = spinnerStyle
	m.spinner.Spinner = spinner.Dot
}

func Launch(endpoints []string) {
	items := []list.Item{}

	for _, endpoint := range endpoints {
		items = append(items, item(endpoint))
	}

	const defaultWidth = 40

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Select the URL you want to check"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	s := spinner.New()
	s.Style = spinnerStyle
	s.Spinner = spinner.Dot

	m := model{
		list:    l,
		spinner: s,
		state:   Listing,
		sub:     make(chan string),
	}

	if _, err := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
