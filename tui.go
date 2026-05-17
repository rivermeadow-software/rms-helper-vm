package main

import (
	"crypto/tls"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type networkMode int

const (
	modeDHCP networkMode = iota
	modeStatic
)

type testProfile struct {
	name        string
	description string
	tests       []troubleshootingTest
}

type troubleshootingTest struct {
	testType string
	port     int
	protocol string
	url      string
}

var testProfiles = []testProfile{
	{
		name:        "RiverMeadow Platform",
		description: "Test connectivity to the RiverMeadow platform",
		tests: []troubleshootingTest{
			{testType: "dnsResolve"},
			{testType: "networkPort", port: 443, protocol: "TCP"},
			{testType: "sslInterception"},
		},
	},
	{
		name:        "Migration Appliance",
		description: "Test connectivity to the migration appliance",
		tests: []troubleshootingTest{
			{testType: "networkPort", port: 8888, protocol: "TCP"},
			{testType: "networkPort", port: 10000, protocol: "TCP"},
			{testType: "networkPort", port: 8080, protocol: "TCP"},
			{testType: "networkPort", port: 443, protocol: "TCP"},
		},
	},
	{
		name:        "Source Worker Appliance",
		description: "Test connectivity to the source worker appliance",
		tests: []troubleshootingTest{
			{testType: "networkPort", port: 5994, protocol: "TCP"},
		},
	},
	{
		name:        "Source Server",
		description: "Test connectivity to the source server",
		tests: []troubleshootingTest{
			{testType: "networkPort", port: 5994, protocol: "TCP"},
		},
	},
	{
		name:        "ICMP Ping",
		description: "Test basic network connectivity with ICMP ping",
		tests: []troubleshootingTest{
			{testType: "icmpPing"},
		},
	},
}

type testStatus int

const (
	running testStatus = iota
	pass
	fail
)

type testResult struct {
	name   string
	status testStatus
	value  string
}

type item struct {
	title string
	desc  string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type customDelegate struct{}

func (d customDelegate) Height() int  { return 2 }
func (d customDelegate) Spacing() int { return 1 }
func (d customDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}
func (d customDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i := listItem.(item)

	cursor := "  "
	if index == m.Index() {
		cursor = "▌ "
	}

	title := itemTitleStyle.Render(i.title)
	if index == m.Index() {
		title = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0A76FD")).
			Bold(true).
			Render(i.title)
	}

	desc := subStyle.Render(i.desc)

	fmt.Fprintf(w, "%s%s\n  %s", cursor, title, desc)
}

type focusArea int

const (
	focusSidebar focusArea = iota
	focusContent
	focusProfiles
	focusNetworkSettings
	focusSystemManagement
)

type model struct {
	list     list.Model
	profiles list.Model
	viewport viewport.Model
	input    textinput.Model
	results  []testResult
	width    int
	height   int
	ready    bool
	inputs   []textinput.Model

	focus               focusArea
	networkMode         networkMode
	networkSelection    int
	networkInputFocus   int
	networkSectionFocus networkSection
	dhcpRunning         bool

	sysMgmtSelection  int
	sysMgmtConfirm    bool
	sysMgmtConfirmSel int

	status string
}

type dhcpResultMsg struct{ err error }
type diagResultMsg struct{ results []testResult }

type networkSection int

const (
	sectionSelector networkSection = iota
	sectionFields
	sectionApply
)

var (
	appStyle = lipgloss.NewStyle().
			Padding(0, 0).
			Foreground(lipgloss.Color("#FFFFFF"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	subStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFFFFF"))

	resultsPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FFFFFF")).
				Padding(1, 2)

	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	footerPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FFFFFF")).
				Padding(0, 1)

	itemTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	passStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")). // green
			Bold(true)

	failStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")). // red
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	focused = lipgloss.NewStyle().Foreground(lipgloss.Color("#0A76FD"))
	blurred = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)
)

func initialModel() model {
	items := []list.Item{
		item{"Show System Information", "Show detailed system information"},
		item{"Network Settings", "Configure network settings"},
		item{"Troubleshooting", "Run diagnostics"},
		item{"System Management", "Restart or shutdown the system"},
	}

	l := list.New(items, customDelegate{}, 0, 0)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)

	inputs := make([]textinput.Model, 5)
	for i := range inputs {
		ti := textinput.New()
		inputs[i] = ti
	}

	inputs[0].Focus()

	profileItems := []list.Item{
		item{"RiverMeadow Platform", "Test connectivity to the RiverMeadow platform"},
		item{"Migration Appliance", "Test connectivity to a migration appliance"},
		item{"Source Worker Appliance", "Test connectivity to a source worker appliance"},
		item{"Source Server", "Test connectivity to a source server"},
		item{"ICMP Ping", "Test basic network connectivity with ICMP ping"},
	}

	profiles := list.New(profileItems, customDelegate{}, 0, 0)
	profiles.SetShowTitle(false)
	profiles.SetShowStatusBar(false)
	profiles.SetShowHelp(false)

	vp := viewport.New(0, 0)

	in := textinput.New()
	in.Placeholder = "Hostname or IP Address"
	in.SetValue("")
	in.CharLimit = 128
	in.Width = 40

	return model{
		results:             []testResult{},
		list:                l,
		profiles:            profiles,
		viewport:            vp,
		input:               in,
		inputs:              inputs,
		focus:               focusSidebar,
		networkMode:         modeDHCP,
		networkSelection:    0,
		networkInputFocus:   0,
		networkSectionFocus: sectionSelector,
		status:              "",
	}
}

func (m model) currentContent() string {
	switch m.list.Index() {
	case 0:
		var ipAddress string
		ifaces, err := net.Interfaces()
		if err == nil {
			for _, iface := range ifaces {
				addrs, err := iface.Addrs()
				if err == nil {
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
							if ipnet.IP.To4() != nil {
								ipAddress = ipnet.IP.String()
								break
							}
						}
					}
				}
				if ipAddress != "" {
					break
				}
			}
		}
		var subnetMask string
		if ipAddress != "" {
			ip := net.ParseIP(ipAddress)
			if ip != nil {
				for _, iface := range ifaces {
					addrs, err := iface.Addrs()
					if err == nil {
						for _, addr := range addrs {
							if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.Equal(ip) {
								mask := ipnet.Mask
								subnetMask = fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
								break
							}
						}
					}
					if subnetMask != "" {
						break
					}
				}
			}
		}

		defaultGateway := getDefaultGateway()
		dnsServers := getDNSServers()
		ipType := getIPAddressType(ipAddress)

		// read /sys/class/dmi/id/product_name for platform info
		productNameBytes, err := os.ReadFile("/sys/class/dmi/id/product_name")
		productName := "Unknown"
		if err == nil {
			productName = strings.TrimSpace(string(productNameBytes))
		}

		return fmt.Sprintf(
			"System Information\n\nPlatform: %s\nIP Address Type: %s\nIP Address: %s\nSubnet Mask: %s\nDefault Gateway: %s\nDNS Servers: %s",
			productName,
			ipType,
			ipAddress,
			subnetMask,
			defaultGateway,
			strings.Join(dnsServers, ", "),
		)
	case 1:
		fieldWidth := m.viewport.Width - 10
		if fieldWidth < 40 {
			fieldWidth = 40
		}

		options := []string{
			"[ ] DHCP",
			"[ ] Static IP",
		}

		if m.networkMode == modeDHCP {
			options[0] = "[x] DHCP"
		}
		if m.networkMode == modeStatic {
			options[1] = "[x] Static IP"
		}

		for i := range options {
			prefix := "  "
			if m.networkSectionFocus == sectionSelector && i == m.networkSelection {
				prefix = "> "
			}
			options[i] = prefix + options[i]
		}

		modeBlock := panelStyle.
			Width(fieldWidth).
			Render(strings.Join(options, "\n"))

		content := "Network Settings\n\n" + modeBlock

		if m.networkMode == modeStatic {
			labels := []string{
				"IP Address",
				"Subnet Mask",
				"Gateway",
				"DNS 1",
				"DNS 2",
			}

			form := "\n\nStatic Configuration\n\n"

			for i := range m.inputs {
				style := blurred
				cursor := " "

				if m.focus == focusNetworkSettings && i == m.networkInputFocus {
					style = focused
					cursor = ">"
				}

				box := inputBoxStyle.Width(fieldWidth - 4).Render(m.inputs[i].View())

				form += fmt.Sprintf("%s %s\n%s\n\n",
					cursor,
					style.Render(labels[i]),
					box,
				)
			}

			content += form
		}
		applyLabel := "[ Apply Settings ]"
		if m.dhcpRunning {
			applyLabel = "[ Requesting... ]"
		}

		if m.networkSectionFocus == sectionApply {
			applyLabel = "> " + applyLabel
		} else {
			applyLabel = "  " + applyLabel
		}

		content += "\n" + focused.Render(applyLabel)

		if m.status != "" {
			content += "\n\n" + statusStyle.Render(m.status)
		}

		return content

	case 2:
		fieldWidth := m.viewport.Width - 12
		if fieldWidth < 40 {
			fieldWidth = 40
		}
		label := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Render("Target")

		box := inputBoxStyle.Width(fieldWidth).Render(
			inputStyle.Render(m.input.View()),
		)

		inputBlock := lipgloss.JoinVertical(
			lipgloss.Left,
			label,
			box,
		)

		profileHeight := len(m.profiles.Items())*4 + 1
		m.profiles.SetSize(fieldWidth-2, profileHeight)
		profilesLabel := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Render("Profiles")
		profilesBlock := panelStyle.
			Width(fieldWidth).
			Height(profileHeight).
			Render(m.profiles.View())

		resultsLabel := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Render("Results")

		resultsHeight := 10

		renderResults := func(res []testResult) string {
			out := ""

			for _, r := range res {
				if r.name == "Running Suite" {
					out += labelStyle.Render(fmt.Sprintf("Profile: %s", r.value)) + "\n\n"
					continue
				}
				var icon string
				switch r.status {
				case pass:
					icon = passStyle.Render("✔")
				case running:
					icon = blurred.Render("…")
				default:
					icon = failStyle.Render("✖")
				}

				out += fmt.Sprintf(
					"%s %s: %s\n",
					icon,
					labelStyle.Render(r.name),
					r.value,
				)
			}

			return out
		}

		resultsBlock := resultsPanelStyle.
			Width(fieldWidth).
			Height(resultsHeight).
			Render(renderResults(m.results))

		status := "Ready"

		if m.focus == focusContent {
			status = "Editing target"
		}
		if m.focus == focusProfiles {
			status = "Selecting profile"
		}

		return inputBlock + "\n\n" +
			profilesLabel + "\n" +
			profilesBlock + "\n\n" +
			resultsLabel + "\n" +
			resultsBlock + "\n\n" +
			"Status: " + status

	case 3:
		fieldWidth := m.viewport.Width - 10
		if fieldWidth < 40 {
			fieldWidth = 40
		}

		actions := []string{"Restart", "Shutdown"}
		actionLines := make([]string, len(actions))
		for i, a := range actions {
			prefix := "  "
			if m.focus == focusSystemManagement && !m.sysMgmtConfirm && i == m.sysMgmtSelection {
				prefix = "> "
			}
			actionLines[i] = prefix + a
		}

		content := "System Management\n\n" +
			panelStyle.Width(fieldWidth).Render(strings.Join(actionLines, "\n"))

		if m.sysMgmtConfirm {
			action := actions[m.sysMgmtSelection]
			warning := failStyle.Render("! " + action + " this system?")
			confirmOpts := []string{
				"[ Yes — " + action + " ]",
				"[ No  — Cancel     ]",
			}
			confirmLines := make([]string, len(confirmOpts))
			for i, opt := range confirmOpts {
				prefix := "  "
				if i == m.sysMgmtConfirmSel {
					prefix = "> "
				}
				confirmLines[i] = prefix + opt
			}
			content += "\n\n" + warning + "\n\n" +
				panelStyle.Width(fieldWidth).Render(strings.Join(confirmLines, "\n"))
		}

		return content

	default:
		return "No content available."
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case dhcpResultMsg:
		m.dhcpRunning = false
		if msg.err != nil {
			m.status = "DHCP failed: no address obtained"
		} else {
			m.status = "Network settings applied"
		}
		return m, tea.Batch(cmds...)

	case diagResultMsg:
		m.results = msg.results
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.focus == focusProfiles {
				target := m.input.Value()
				if target == "" {
					m.results = []testResult{
						{name: "Error", status: fail, value: "No target specified"},
					}
					break
				}

				selectedProfile := m.profiles.SelectedItem().(item).title
				m.results = pendingResults(selectedProfile)
				profile := selectedProfile
				cmds = append(cmds, func() tea.Msg {
					results := []testResult{
						{name: "Running Suite", status: pass, value: profile},
					}
					results = append(results, runDiagnostics(target, profile)...)
					return diagResultMsg{results: results}
				})

			}
		case "tab":
			if m.list.Index() == 1 {
				if m.focus == focusSidebar {
					m.focus = focusNetworkSettings
					m.status = ""
				} else {
					m.focus = focusSidebar
					m.status = ""
				}
			}
			if m.list.Index() == 2 {
				switch m.focus {
				case focusSidebar:
					m.focus = focusContent
					m.input.Focus()
				case focusContent:
					m.focus = focusProfiles
					m.input.Blur()
				case focusProfiles:
					m.focus = focusSidebar
				}
			}
			if m.list.Index() == 3 {
				if m.focus == focusSidebar {
					m.focus = focusSystemManagement
				} else {
					m.focus = focusSidebar
					m.sysMgmtConfirm = false
				}
			}
		}
	}

	if m.list.Index() == 3 && m.focus == focusSystemManagement {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up":
				if m.sysMgmtConfirm {
					if m.sysMgmtConfirmSel > 0 {
						m.sysMgmtConfirmSel--
					}
				} else if m.sysMgmtSelection > 0 {
					m.sysMgmtSelection--
				}
			case "down":
				if m.sysMgmtConfirm {
					if m.sysMgmtConfirmSel < 1 {
						m.sysMgmtConfirmSel++
					}
				} else if m.sysMgmtSelection < 1 {
					m.sysMgmtSelection++
				}
			case "enter", " ":
				if !m.sysMgmtConfirm {
					m.sysMgmtConfirm = true
					m.sysMgmtConfirmSel = 1 // default to "No"
				} else if m.sysMgmtConfirmSel == 0 {
					if m.sysMgmtSelection == 0 {
						_ = exec.Command("reboot").Run()
					} else {
						_ = exec.Command("poweroff").Run()
					}
				} else {
					m.sysMgmtConfirm = false
				}
			case "esc", "n", "N":
				m.sysMgmtConfirm = false
			}
		}
		return m, tea.Batch(cmds...)
	}

	if m.list.Index() == 1 && m.focus == focusNetworkSettings {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {

			case "up":
				m.status = ""
				if m.networkSectionFocus == sectionApply {
					if m.networkMode == modeStatic {
						m.networkSectionFocus = sectionFields
						m.networkInputFocus = len(m.inputs) - 1
					} else {
						m.networkSectionFocus = sectionSelector
					}
				} else if m.networkSectionFocus == sectionSelector {
					if m.networkSelection > 0 {
						m.networkSelection--
					}
				} else {
					if m.networkInputFocus == 0 {
						m.networkSectionFocus = sectionSelector
					} else {
						m.networkInputFocus--
					}
				}

			case "down":
				m.status = ""
				if m.networkSectionFocus == sectionSelector {
					if m.networkSelection < 1 {
						m.networkSelection++
					} else if m.networkMode == modeStatic {
						m.networkSectionFocus = sectionFields
					} else {
						m.networkSectionFocus = sectionApply
					}
				} else if m.networkSectionFocus == sectionFields {
					if m.networkInputFocus < len(m.inputs)-1 {
						m.networkInputFocus++
					} else {
						m.networkSectionFocus = sectionApply
					}
				}

			case "enter", " ":
				if m.networkSectionFocus == sectionSelector {
					if m.networkSelection == 0 {
						m.networkMode = modeDHCP
					} else {
						m.networkMode = modeStatic
						m.networkInputFocus = 0
					}
				} else if m.networkSectionFocus == sectionApply {
					if m.networkMode == modeDHCP {
						if m.dhcpRunning {
							break
						}
						m.dhcpRunning = true
						m.status = "Requesting DHCP address..."
						cmds = append(cmds, func() tea.Msg {
							return dhcpResultMsg{err: applyDHCP()}
						})
					} else {
						err := applyStatic(
							m.inputs[0].Value(),
							m.inputs[1].Value(),
							m.inputs[2].Value(),
							m.inputs[3].Value(),
							m.inputs[4].Value(),
						)
						if err != nil {
							m.status = "Failed: " + err.Error()
						} else {
							m.status = "Network settings applied"
						}
					}
				}
			}
		}

		if m.networkMode == modeStatic && m.networkSectionFocus == sectionFields {
			for i := range m.inputs {
				if i == m.networkInputFocus {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}

				var cmd tea.Cmd
				oldVal := m.inputs[i].Value()

				m.inputs[i], cmd = m.inputs[i].Update(msg)

				if m.inputs[i].Value() != oldVal {
					m.status = ""
				}

				cmds = append(cmds, cmd)
			}
		}

		return m, tea.Batch(cmds...)
	}

	if m.focus == focusContent && m.list.Index() == 2 {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

	} else if m.focus == focusProfiles && m.list.Index() == 2 {
		var cmd tea.Cmd
		m.profiles, cmd = m.profiles.Update(msg)
		cmds = append(cmds, cmd)

	} else {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	usableWidth := m.width
	usableHeight := m.height

	header := headerStyle.Width(usableWidth).Render("RiverMeadow Troubleshooting Appliance")
	headerHeight := lipgloss.Height(header)

	footerReservedHeight := 3
	contentHeight := usableHeight - headerHeight - footerReservedHeight
	if contentHeight < 10 {
		contentHeight = 10
	}

	panelFrameW := panelStyle.GetHorizontalFrameSize()
	panelFrameH := panelStyle.GetVerticalFrameSize()

	leftTotal := usableWidth / 3
	rightTotal := usableWidth - leftTotal

	leftContentW := leftTotal - panelFrameW
	rightContentW := rightTotal - panelFrameW

	contentInnerH := contentHeight - panelFrameH

	m.list.SetSize(leftContentW, contentInnerH)

	m.viewport.Width = rightContentW
	m.viewport.Height = contentInnerH
	m.viewport.SetContent(
		contentStyle.Width(rightContentW).Render(m.currentContent()),
	)

	leftPane := panelStyle.
		Width(leftContentW).
		Height(contentInnerH).
		Render(m.list.View())

	rightPane := panelStyle.
		Width(rightContentW).
		Height(contentInnerH).
		Render(m.viewport.View())

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPane,
		rightPane,
	)

	bodyWidth := lipgloss.Width(body)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0A76FD")).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA"))

	// individual hints
	hints := []string{
		keyStyle.Render("[Tab]") + descStyle.Render(" Switch Focus"),
		keyStyle.Render("[Enter]") + descStyle.Render(" Select/Run"),
		keyStyle.Render("[ctrl+c]") + descStyle.Render(" Force Quit"),
	}

	// small spacing between hints
	line := strings.Join(hints, "   ") // 👈 tweak spacing here

	footer := footerPanelStyle.
		Width(bodyWidth - 2).
		//Render("[Tab] Switch Focus  [↑↓] Navigate  [q] Quit")
		Render(line)

	layout := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		footer,
	)

	return appStyle.Render(layout)
}

func tui() {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func testDisplayName(t troubleshootingTest) string {
	switch t.testType {
	case "dnsResolve":
		return "DNS Resolution"
	case "networkPort":
		return fmt.Sprintf("TCP (%d)", t.port)
	case "sslInterception":
		return "SSL Issuer Check"
	case "icmpPing":
		return "ICMP Ping"
	default:
		return t.testType
	}
}

func pendingResults(profile string) []testResult {
	results := []testResult{
		{name: "Running Suite", status: pass, value: profile},
	}
	for _, p := range testProfiles {
		if p.name == profile {
			for _, t := range p.tests {
				results = append(results, testResult{
					name:   testDisplayName(t),
					status: running,
					value:  "pending...",
				})
			}
		}
	}
	return results
}

func runDiagnostics(target string, profile string) []testResult {
	results := []testResult{}

	for _, p := range testProfiles {
		if p.name == profile {
			// Add additional standard tests for RiverMeadow platform profile
			if p.name == "RiverMeadow Platform" && target == "migrate.rivermeadow.com" {
				backendHosts := []string{"52.9.247.1", "52.9.142.11"}
				for _, host := range backendHosts {
					conn, err := net.DialTimeout("tcp", host+":443", 2*time.Second)
					if err != nil {
						results = append(results, testResult{
							name:   fmt.Sprintf("TCP (%s:443)", host),
							status: fail,
							value:  "unreachable",
						})
					} else {
						conn.Close()
						results = append(results, testResult{
							name:   fmt.Sprintf("TCP (%s:443)", host),
							status: pass,
							value:  "open",
						})
					}
				}
			}
			for _, t := range p.tests {
				switch t.testType {
				case "dnsResolve":
					// DNS
					ips, err := net.LookupHost(target)
					if err != nil {
						results = append(results, testResult{
							name:   "DNS Resolution",
							status: fail,
							value:  err.Error(),
						})
					} else {
						results = append(results, testResult{
							name:   "DNS Resolution",
							status: pass,
							value:  fmt.Sprintf("%v", ips),
						})
					}
				case "networkPort":
					// TCP
					conn, err := net.DialTimeout("tcp", target+":"+fmt.Sprintf("%d", t.port), 2*time.Second)
					if err != nil {
						results = append(results, testResult{
							name:   fmt.Sprintf("TCP (%d)", t.port),
							status: fail,
							value:  "unreachable",
						})
					} else {
						conn.Close()
						results = append(results, testResult{
							name:   fmt.Sprintf("TCP (%d)", t.port),
							status: pass,
							value:  "open",
						})
					}
				case "sslInterception":
					// Run SSL interception test
					output := sslIssuerTest(target, "Amazon")
					results = append(results, output)
				case "icmpPing":
					result := Ping(target, 4, 2*time.Second)
					var outStatus testStatus
					if result.Success {
						outStatus = pass
					} else {
						outStatus = fail
					}
					results = append(results, testResult{
						name:   "ICMP Ping",
						status: outStatus,
						value:  result.Output,
					})
				default:
					fmt.Printf("Unknown test type: %s\n", t.testType)
				}
			}
		}
	}

	return results
}

func sslIssuerTest(target string, expectedIssuer string) testResult {
	dialer := &tls.Dialer{
		Config: &tls.Config{
			InsecureSkipVerify: true, // we only inspect certs, not validate trust
		},
	}

	conn, err := dialer.Dial("tcp", target+":443")
	if err != nil {
		return testResult{
			name:   "SSL Issuer Check",
			status: fail,
			value:  "connection failed: " + err.Error(),
		}
	}
	defer conn.Close()

	state := conn.(*tls.Conn).ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return testResult{
			name:   "SSL Issuer Check",
			status: fail,
			value:  "no certificates returned",
		}
	}

	leaf := state.PeerCertificates[0]

	actualIssuer := formatName(leaf.Issuer)

	// Normalize comparison for easier matching
	match := strings.Contains(strings.ToLower(actualIssuer), strings.ToLower(expectedIssuer))

	if match {
		return testResult{
			name:   "SSL Issuer Check",
			status: pass,
			value:  fmt.Sprintf("issuer OK: %s", actualIssuer),
		}
	}

	return testResult{
		name:   "SSL Issuer Check",
		status: fail,
		value: fmt.Sprintf(
			"POSSIBLE SSL INTERCEPTION (expected: %s - actual: %s)",
			expectedIssuer,
			actualIssuer,
		),
	}
}

func formatName(name pkix.Name) string {
	if len(name.Organization) > 0 {
		return strings.Join(name.Organization, ", ")
	}
	if name.CommonName != "" {
		return name.CommonName
	}
	return "unknown issuer"
}

func applyDHCP() error {
	// Kill any existing udhcpc on this interface so its lease file doesn't
	// block a fresh attempt after a previous failure.
	_, _ = exec.Command("killall", "-q", "udhcpc").CombinedOutput()
	_, _ = exec.Command("ip", "addr", "flush", "dev", "eth0").CombinedOutput()
	_, _ = exec.Command("ip", "link", "set", "eth0", "up").CombinedOutput()

	cmd := exec.Command("udhcpc", "-i", "eth0", "-q", "-t", "2", "-n")
	if err := cmd.Run(); err != nil {
		return err
	}
	_ = os.WriteFile(networkStateFile, []byte("DHCP"), 0644)
	return nil
}

func applyStatic(ip, mask, gw, dns1, dns2 string) error {
	_, _ = exec.Command("ip", "addr", "flush", "dev", "eth0").CombinedOutput()

	cidr, err := maskToCIDR(mask)
	if err != nil {
		return err
	}

	out, err := exec.Command("ip", "addr", "add", ip+"/"+cidr, "dev", "eth0").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip addr add failed: %v: %s", err, string(out))
	}

	out, err = exec.Command("ip", "route", "replace", "default", "via", gw).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip route failed: %v: %s", err, string(out))
	}

	resolv := "nameserver " + dns1 + "\n"
	if dns2 != "" {
		resolv += "nameserver " + dns2 + "\n"
	}

	if err := os.WriteFile("/etc/resolv.conf", []byte(resolv), 0644); err != nil {
		return err
	}

	_ = os.WriteFile(networkStateFile, []byte("Static"), 0644)
	return nil
}

func maskToCIDR(mask string) (string, error) {
	ip := net.ParseIP(mask).To4()
	if ip == nil {
		return "", fmt.Errorf("invalid subnet mask")
	}

	ones, bitsTotal := net.IPMask(ip).Size()
	if bitsTotal != 32 {
		return "", fmt.Errorf("invalid subnet mask size")
	}

	return strconv.Itoa(ones), nil
}

func getDefaultGateway() string {
	out, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return ""
	}

	fields := strings.Fields(string(out))

	for i, f := range fields {
		if f == "via" && i+1 < len(fields) {
			return fields[i+1]
		}
	}

	return ""
}

const networkStateFile = "/tmp/networkstate"

func getIPAddressType(ipAddress string) string {
	if ipAddress == "" {
		return "Unknown"
	}
	data, err := os.ReadFile(networkStateFile)
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(data))
}

func getDNSServers() []string {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}

	var servers []string

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "nameserver") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				servers = append(servers, fields[1])
			}
		}
	}

	return servers
}
