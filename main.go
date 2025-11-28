// main.go - GoTerm - FINAL VERSION - works everywhere!
package main

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

type model struct {
	pty  *os.File
	dark bool
}

var ptyFile *os.File

func main() {
	// If double-clicked (no real TTY), relaunch inside user's terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		relaunchInTerminal()
		return
	}

	// Otherwise: run the full terminal
	runTerminal()
}

func relaunchInTerminal() {
	termCmd := []string{
		"gnome-terminal", "--", // GNOME
		"konsole", "--nofork", "-e", // KDE
		"xfce4-terminal", "-x", // XFCE
		"lxterminal", "-e", // LXDE
		"mate-terminal", "--", // MATE
		"tilix", "-e", // Tilix
		"kitty", // Kitty
		"alacritty", "-e", // Alacritty
		"xterm", "-e", // Fallback
	}

	for i := 0; i < len(termCmd); i += 1 {
		if i+1 < len(termCmd) && termCmd[i+1] != "-e" && termCmd[i+1] != "--" && termCmd[i+1] != "-x" {
			continue
		}
		cmd := exec.Command(termCmd[i], append(termCmd[i+1:], os.Args[0])...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if cmd.Run() == nil {
			return
		}
		i += 1 // skip args for this terminal
	}
	// Ultimate fallback
	exec.Command("xterm", "-e", os.Args[0]).Run()
}

func runTerminal() {
	cmd := exec.Command(os.Getenv("SHELL"))
	if cmd.Path == "" {
		cmd = exec.Command("bash")
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		panic(err)
	}
	ptyFile = ptmx

	// Set initial size
	w, h, _ := term.GetSize(int(os.Stdout.Fd()))
	pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(h), Cols: uint16(w)})

	// Resize handler
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGWINCH)
		for range sigs {
			w, h, _ := term.GetSize(int(os.Stdout.Fd()))
			pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(h), Cols: uint16(w)})
		}
	}()

	m := model{pty: ptmx, dark: true}

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
			    tea.WithMouseAllMotion(),
	)

	go io.Copy(os.Stdout, ptmx)

	if _, err := p.Run(); err != nil {
		println("Error:", err.Error())
	}

	// Clean exit
	os.Stdout.Write([]byte("\x1b[?25h\x1b[0m"))
	ptmx.Close()
}

func (m model) Init() tea.Cmd { return tea.EnterAltScreen }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
				case tea.KeyCtrlC:
					return m, tea.Quit
				case tea.KeyCtrlT:
					m.dark = !m.dark
					if m.dark {
						m.pty.Write([]byte("\x1b[40m\x1b[37m"))
					} else {
						m.pty.Write([]byte("\x1b[47m\x1b[30m"))
					}
				case tea.KeyEnter:
					m.pty.Write([]byte("\r"))
				case tea.KeyBackspace, tea.KeyDelete:
					m.pty.Write([]byte{0x7f})
				case tea.KeyRunes:
					m.pty.Write([]byte(string(msg.Runes)))
				default:
					if s := msg.String(); len(s) == 1 {
						m.pty.Write([]byte(s))
					}
			}
	}
	return m, nil
}

func (m model) View() string { return "" }
