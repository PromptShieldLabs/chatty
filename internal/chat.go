package internal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/stig/goaichat/internal/config"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
)

// Session manages a chat conversation with history.
type Session struct {
	client    *Client
	config    *config.Config
	history   []Message
	input     io.Reader
	output    io.Writer
	useColors bool
}

// NewSession creates a new chat session.
func NewSession(client *Client, cfg *config.Config) (*Session, error) {
	if client == nil {
		return nil, errors.New("client cannot be nil")
	}
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	return &Session{
		client:    client,
		config:    cfg,
		history:   make([]Message, 0, 16),
		input:     os.Stdin,
		output:    os.Stdout,
		useColors: true,
	}, nil
}

// Run starts the interactive chat loop.
func (s *Session) Run(ctx context.Context) error {
	if s == nil {
		return errors.New("session is nil")
	}
	if ctx == nil {
		return errors.New("context is nil")
	}

	s.printWelcome()

	scanner := bufio.NewScanner(s.input)

	for {
		s.printPrompt()

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return nil
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			exit, err := s.handleCommand(input)
			if err != nil {
				s.printError(err.Error())
			}
			if exit {
				return nil
			}
			continue
		}

		// Send message to AI
		if err := s.sendMessage(ctx, input); err != nil {
			s.printError(err.Error())
			continue
		}
	}
}

func (s *Session) sendMessage(ctx context.Context, input string) error {
	// Add user message to history
	userMsg := Message{Role: "user", Content: input}
	s.history = append(s.history, userMsg)

	// Call API
	reply, err := s.client.Chat(ctx, s.history, s.config.Model.Name, s.config.Model.Temperature)
	if err != nil {
		// Remove the user message if the request failed
		s.history = s.history[:len(s.history)-1]
		return fmt.Errorf("chat request failed: %w", err)
	}

	// Add assistant response to history
	assistantMsg := Message{Role: "assistant", Content: reply}
	s.history = append(s.history, assistantMsg)

	// Print response
	s.printAssistant(reply)

	return nil
}

func (s *Session) handleCommand(cmd string) (exit bool, err error) {
	switch cmd {
	case "/exit", "/quit":
		s.println(s.colorize(colorYellow, "Goodbye!"))
		return true, nil

	case "/reset", "/clear":
		s.history = s.history[:0]
		s.println(s.colorize(colorYellow, "History cleared."))
		return false, nil

	case "/help":
		s.printHelp()
		return false, nil

	case "/history":
		s.printHistory()
		return false, nil

	default:
		return false, fmt.Errorf("unknown command %q. Try /help", cmd)
	}
}

func (s *Session) printWelcome() {
	s.println(s.colorize(colorCyan, "=== Goaichat ==="))
	s.println(fmt.Sprintf("Model: %s | Temperature: %.1f", s.config.Model.Name, s.config.Model.Temperature))
	s.println(s.colorize(colorYellow, "Type /help for commands, /exit to quit"))
	s.println("")
}

func (s *Session) printHelp() {
	help := `Available commands:
  /help     - Show this help message
  /exit     - Exit the chat
  /reset    - Clear conversation history
  /history  - Show conversation history`
	s.println(s.colorize(colorYellow, help))
}

func (s *Session) printHistory() {
	if len(s.history) == 0 {
		s.println(s.colorize(colorYellow, "No history yet."))
		return
	}

	s.println(s.colorize(colorYellow, "=== History ==="))
	for i, msg := range s.history {
		prefix := "User"
		color := colorCyan
		if msg.Role == "assistant" {
			prefix = "AI"
			color = colorGreen
		}
		s.println(fmt.Sprintf("%s[%d] %s:%s %s", s.colorize(colorYellow, ""), i+1, prefix, colorReset, s.colorize(color, msg.Content)))
	}
}

func (s *Session) printPrompt() {
	fmt.Fprint(s.output, s.colorize(colorCyan, "> "))
}

func (s *Session) printAssistant(text string) {
	s.println(s.colorize(colorGreen, text))
}

func (s *Session) printError(text string) {
	s.println(s.colorize(colorRed, "Error: "+text))
}

func (s *Session) println(text string) {
	fmt.Fprintln(s.output, text)
}

func (s *Session) colorize(color, text string) string {
	if !s.useColors {
		return text
	}
	return color + text + colorReset
}

// SetIO overrides input/output streams (useful for testing).
func (s *Session) SetIO(in io.Reader, out io.Writer) {
	if in != nil {
		s.input = in
	}
	if out != nil {
		s.output = out
	}
}

// DisableColors turns off ANSI color output.
func (s *Session) DisableColors() {
	s.useColors = false
}
