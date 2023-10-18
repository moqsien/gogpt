package gpt

import (
	"github.com/moqsien/gogpt/pkgs/config"
	"github.com/sashabaranov/go-openai"
)

/*
Manage Chatgpt conversation
*/

type QuesAnsw struct {
	Q string // question
	A string // answer
}

type Conversation struct {
	Context []QuesAnsw
	History []QuesAnsw
	Current *QuesAnsw
	Tokens  int
	CNF     *config.Config
	Prompt  *GPTPrompt
}

func NewConversation(cnf *config.Config) (conv *Conversation) {
	conv = &Conversation{
		Context: []QuesAnsw{},
		History: []QuesAnsw{},
		CNF:     cnf,
	}
	conv.Prompt = NewGPTPrompt(cnf)
	return
}

func (that *Conversation) AddQuestion(ques string) {
	that.Current = &QuesAnsw{
		Q: ques,
	}
	that.Tokens = 0
}

func (that *Conversation) AddAnswer(answ string, completed bool) {
	if that.Current == nil {
		return
	}
	that.Current.A += answ
	if completed {
		that.Context = append(that.Context, *that.Current)
		that.Tokens = 0
		if len(that.Context) > that.CNF.OpenAI.ContextLen {
			that.History = append(that.History, that.Context[0])
			that.Context = that.Context[1:]
		}
		that.Current = nil
	}
}

func (that *Conversation) GetMessages() []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, 2*len(that.Context)+2)
	messages = append(
		messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: that.Prompt.PromptStr(),
		},
	)
	for _, c := range that.Context {
		messages = append(
			messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: c.Q,
			},
		)
		messages = append(
			messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: c.A,
			},
		)
	}
	if that.Current != nil {
		messages = append(
			messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: that.Current.Q,
			},
		)
	}
	return messages
}

func (that *Conversation) GetTokens() int {
	if that.Tokens == 0 {
		that.Tokens = NumTokensFromMessages(that.GetMessages(), that.CNF.OpenAI.Model)
	}
	return that.Tokens
}

func (that *Conversation) ClearContext() {
	that.History = append(that.History, that.Context...)
	that.Context = nil
	that.Tokens = 0
}

func (that *Conversation) CurrentAnswer() string {
	if that.Current == nil {
		return ""
	}
	return that.Current.A
}

func (that *Conversation) LastAnswer() string {
	if len(that.Context) == 0 {
		return ""
	}
	return that.Context[len(that.Context)-1].A
}

func (that *Conversation) Len() int {
	l := len(that.History) + len(that.Context)
	if that.Current != nil {
		l++
	}
	return l
}

func (that *Conversation) GetQuestionByIndex(idx int) string {
	if idx < 0 || idx >= that.Len() {
		return ""
	}
	if idx < len(that.History) {
		return that.History[idx].Q
	}
	return that.Context[idx-len(that.History)].Q
}