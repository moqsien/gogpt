package tui

import (
	"io"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/moqsien/gogpt/pkgs/config"
	"github.com/moqsien/gogpt/pkgs/gpt"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
)

type AnswerContinue string

type ConversationModel struct {
	Viewport     viewport.Model
	TextArea     textarea.Model
	Spinner      spinner.Model
	R            *glamour.TermRenderer
	CNF          *config.Config
	WindowHeight int
	WindowWidth  int
	GPT          *gpt.GPT
	Conversation *gpt.Conversation
	Receiving    bool
}

func NewConversationModel(cnf *config.Config) (cvm *ConversationModel) {
	cvm = &ConversationModel{
		CNF:          cnf,
		GPT:          gpt.NewGPT(cnf),
		Conversation: gpt.NewConversation(cnf),
	}
	cvm.Spinner = spinner.New(spinner.WithSpinner(spinner.Meter))
	cvm.TextArea = textarea.New()
	cvm.TextArea.Cursor.SetMode(cursor.CursorBlink)
	cvm.TextArea.Placeholder = "enter you message"
	cvm.TextArea.CharLimit = -1
	cvm.TextArea.FocusedStyle.CursorLine = lipgloss.NewStyle()
	cvm.TextArea.ShowLineNumbers = false
	cvm.TextArea.Focus()
	cvm.TextArea.CursorEnd()
	cvm.TextArea.SetHeight(2)
	cvm.Viewport = viewport.Model{}
	cvm.R, _ = glamour.NewTermRenderer(
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(0),
	)
	return
}

func (that *ConversationModel) Init() tea.Cmd {
	return tea.Batch(that.Spinner.Tick, textarea.Blink)
}

// TODO: keymap & viewpord render & help info & logfile
func (that *ConversationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		that.WindowWidth = msg.Width
		that.WindowHeight = msg.Height
		that.TextArea.SetWidth(that.WindowWidth)
		that.Viewport.Width = msg.Width - 5
		that.Viewport.MouseWheelEnabled = true
		that.Viewport.Height = msg.Height - that.TextArea.Height() - lipgloss.Height(that.Spinner.View()) - lipgloss.Height(lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Render("title\n"))
	case spinner.TickMsg:
		if that.Receiving {
			that.Spinner, cmd = that.Spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	case tea.KeyMsg:
		switch keyPress := msg.String(); keyPress {
		case "enter":
			messageStr := that.TextArea.Value()
			that.TextArea.Reset()
			that.TextArea.Blur()
			if messageStr != "" {
				that.Conversation.AddQuestion(messageStr)
				msgList := that.Conversation.GetMessages()
				that.Receiving = true
				answerStr, err := that.GPT.SendMsg(msgList)
				if err == io.EOF {
					that.Receiving = false
				} else {
					cmds = append(cmds, func() tea.Msg {
						var msg AnswerContinue
						return msg
					})
				}
				that.Conversation.AddAnswer(answerStr, !that.Receiving)
				that.Viewport.SetContent(that.RenderQA(*that.Conversation.Current))
				that.Viewport.GotoBottom()
			}
		case "up", "down":
			that.Viewport, cmd = that.Viewport.Update(msg)
			cmds = append(cmds, cmd)
		case "ctrl+p":
			qa := that.Conversation.GetPrevQA()
			that.Viewport.SetContent(that.RenderQA(qa))
		case "ctrl+f":
			qa := that.Conversation.GetNextQA()
			that.Viewport.SetContent(that.RenderQA(qa))
		default:
			if !that.TextArea.Focused() && !that.Receiving {
				cmd = that.TextArea.Focus()
				cmds = append(cmds, cmd)
			}
			that.TextArea, cmd = that.TextArea.Update(msg)
			cmds = append(cmds, cmd)
		}
	case AnswerContinue:
		answerStr, err := that.GPT.RecvMsg()
		if err == io.EOF {
			that.Receiving = false
		} else {
			cmds = append(cmds, func() tea.Msg {
				var msg AnswerContinue
				return msg
			})
		}
		that.Conversation.AddAnswer(answerStr, !that.Receiving)
		if that.Conversation.Current != nil {
			that.Viewport.SetContent(that.RenderQA(*that.Conversation.Current))
			that.Viewport.GotoBottom()
		}
	}
	return that, tea.Batch(cmds...)
}

func (that *ConversationModel) ContainsCJK(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Han, unicode.Hangul, unicode.Hiragana, unicode.Katakana) {
			return true
		}
	}
	return false
}

func (that *ConversationModel) EnsureTrailingNewline(s string) string {
	if !strings.HasSuffix(s, "\n") {
		return s + "\n"
	}
	return s
}

var (
	senderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	botStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	// errorStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
	// footerStyle = lipgloss.NewStyle().
	// 		Height(1).
	// 		BorderTop(true).
	// 		BorderStyle(lipgloss.NormalBorder()).
	// 		BorderForeground(lipgloss.Color("8")).
	// 		Faint(true)
)

func (that *ConversationModel) RenderQA(qa gpt.QuesAnsw) string {
	var (
		b       strings.Builder
		content string
	)
	b.WriteString(senderStyle.Render("You: "))

	content = qa.Q
	if that.ContainsCJK(content) {
		content = wrap.String(content, that.WindowWidth-5)
	} else {
		content = wordwrap.String(content, that.WindowWidth-5)
	}
	content, _ = that.R.Render(content)
	b.WriteString(that.EnsureTrailingNewline(content))

	b.WriteString(botStyle.Render("Bot: "))
	content = qa.A
	if that.ContainsCJK(content) {
		content = wrap.String(content, that.WindowWidth-5)
	} else {
		content = wordwrap.String(content, that.WindowWidth-5)
	}
	content, _ = that.R.Render(content)
	b.WriteString(that.EnsureTrailingNewline(content))
	return b.String()
}

func (that *ConversationModel) View() string {
	if that.WindowWidth == 0 || that.WindowHeight == 0 {
		return "Initializing..."
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		that.Viewport.View(),
		that.TextArea.View(),
		that.Spinner.View(),
	)
}
