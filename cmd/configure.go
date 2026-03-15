package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	iclient "github.com/tomdiekmann/icu/internal/client"
	"github.com/tomdiekmann/icu/internal/config"
)

func init() {
	rootCmd.AddCommand(configureCmd)
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure your Intervals.icu API key interactively",
	RunE:  runConfigure,
}

func runConfigure(cmd *cobra.Command, args []string) error {
	p := tea.NewProgram(newConfigureModel())
	m, err := p.Run()
	if err != nil {
		return err
	}

	model := m.(configureModel)
	if model.cancelled {
		fmt.Fprintln(os.Stderr, "cancelled")
		return nil
	}
	if model.err != nil {
		return model.err
	}

	// Save config
	cfg := &config.Config{
		APIKey:        strings.TrimSpace(model.input.Value()),
		AthleteID:     "0",
		DefaultOutput: "auto",
		Units:         "metric",
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	path, _ := config.ConfigFilePath()
	fmt.Printf("\nConfig saved to %s\n", path)
	return nil
}

type configureModel struct {
	input     textinput.Model
	validated bool
	athlete   string
	err       error
	cancelled bool
	loading   bool
}

type validateResult struct {
	name string
	err  error
}

func newConfigureModel() configureModel {
	ti := textinput.New()
	ti.Placeholder = "your_api_key_here"
	ti.Focus()
	ti.Width = 50
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'

	return configureModel{input: ti}
}

func (m configureModel) Init() tea.Cmd {
	return textinput.Blink
}

type validateMsg validateResult

func validateKey(key string) tea.Cmd {
	return func() tea.Msg {
		c := iclient.New(strings.TrimSpace(key), "0", false)
		data, err := c.Get("/api/v1/athlete/0")
		if err != nil {
			return validateMsg{err: err}
		}
		var obj struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(data, &obj)
		name := obj.Name
		if name == "" {
			name = "athlete"
		}
		return validateMsg{name: name}
	}
}

func (m configureModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.loading {
				return m, nil
			}
			key := strings.TrimSpace(m.input.Value())
			if key == "" {
				return m, nil
			}
			m.loading = true
			return m, validateKey(key)
		}
	case validateMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.validated = true
		m.athlete = msg.name
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m configureModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("icu configure") + "\n\n")
	b.WriteString("Enter your Intervals.icu API key\n")
	b.WriteString(dimStyle.Render("(Settings > Developer Settings in the Intervals.icu web app)") + "\n\n")
	b.WriteString(m.input.View() + "\n\n")

	if m.loading {
		b.WriteString(dimStyle.Render("Validating API key...") + "\n")
	} else if m.validated {
		b.WriteString(successStyle.Render(fmt.Sprintf("✓ Authenticated as %s", m.athlete)) + "\n")
	} else if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("✗ %s", m.err.Error())) + "\n")
	} else {
		b.WriteString(dimStyle.Render("Press Enter to validate • Esc to cancel") + "\n")
	}

	return b.String()
}
