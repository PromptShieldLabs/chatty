package internal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/PromptShieldLabs/chatty/internal/config"
	"github.com/charmbracelet/glamour"
)

// ANSI color codes and styles for terminal output
const (
	colorReset   = "\033[0m"
	colorCyan    = "\033[36m"
	colorGreen   = "\033[32m"
	colorRed     = "\033[31m"
	colorYellow  = "\033[33m"
	colorMagenta = "\033[35m"
	styleDim     = "\033[2m"
	styleItalic  = "\033[3m"
)

// Session manages a chat conversation with history.
type Session struct {
	client       *Client
	config       *config.Config
	history      []Message
	input        io.Reader
	output       io.Writer
	useColors    bool
	version      string
	mdRenderer   *glamour.TermRenderer
	renderMarkdown bool
}

// NewSession creates a new chat session.
func NewSession(client *Client, cfg *config.Config, version string) (*Session, error) {
	if client == nil {
		return nil, errors.New("client cannot be nil")
	}
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	// Initialize markdown renderer with dark style
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create markdown renderer: %w", err)
	}

	return &Session{
		client:         client,
		config:         cfg,
		history:        make([]Message, 0, 16),
		input:          os.Stdin,
		output:         os.Stdout,
		useColors:      true,
		version:        version,
		mdRenderer:     renderer,
		renderMarkdown: true,
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

	var reply string
	var err error

	if s.config.Model.Stream {
		// Streaming mode
		reply, err = s.streamResponse(ctx)
	} else {
		// Non-streaming mode
		reply, err = s.client.Chat(ctx, s.history, s.config.Model.Name, s.config.Model.Temperature)
		if err == nil {
			s.printAssistant(reply)
		}
	}

	if err != nil {
		// Remove the user message if the request failed
		s.history = s.history[:len(s.history)-1]
		return fmt.Errorf("chat request failed: %w", err)
	}

	// Add assistant response to history
	assistantMsg := Message{Role: "assistant", Content: reply}
	s.history = append(s.history, assistantMsg)

	return nil
}

func (s *Session) streamResponse(ctx context.Context) (string, error) {
	var fullResponse strings.Builder
	var buffer strings.Builder
	var afterThinkingContent strings.Builder
	inThinking := false
	thinkingStarted := false
	thinkingClosed := false

	// Regex patterns for thinking tags
	thinkTagPattern := regexp.MustCompile(`<think>|<thinking>`)
	thinkClosePattern := regexp.MustCompile(`</think>|</thinking>`)

	err := s.client.ChatStream(ctx, s.history, s.config.Model.Name, s.config.Model.Temperature, func(chunk string) error {
		fullResponse.WriteString(chunk)
		
		// If we're past thinking tags, collect for markdown rendering
		if thinkingClosed {
			afterThinkingContent.WriteString(chunk)
			return nil
		}
		
		buffer.WriteString(chunk)
		bufferStr := buffer.String()

		// Check for opening thinking tags
		if !inThinking && thinkTagPattern.MatchString(bufferStr) {
			loc := thinkTagPattern.FindStringIndex(bufferStr)
			if loc != nil {
				// Print content before tag
				beforeTag := bufferStr[:loc[0]]
				if beforeTag != "" && !thinkingStarted {
					if s.useColors {
						fmt.Fprint(s.output, colorGreen)
					}
					fmt.Fprint(s.output, beforeTag)
				}
				
				// Switch to thinking mode
				inThinking = true
				thinkingStarted = true
				if s.useColors {
					fmt.Fprint(s.output, colorReset+styleDim+colorMagenta)
				}
				
				// Print opening tag and content after it
				afterTag := bufferStr[loc[0]:]
				fmt.Fprint(s.output, afterTag)
				buffer.Reset()
			}
		} else if inThinking && thinkClosePattern.MatchString(bufferStr) {
			// Check for closing thinking tags
			loc := thinkClosePattern.FindStringIndex(bufferStr)
			if loc != nil {
				// Print content including closing tag
				upToAndIncludingTag := bufferStr[:loc[1]]
				fmt.Fprint(s.output, upToAndIncludingTag)
				
				// Switch back to normal mode
				inThinking = false
				thinkingClosed = true
				if s.useColors {
					fmt.Fprint(s.output, colorReset)
				}
				
				// Start collecting content after closing tag for markdown
				afterTag := bufferStr[loc[1]:]
				afterThinkingContent.WriteString(afterTag)
				buffer.Reset()
			}
		} else {
			// Normal streaming - print as we go
			if !thinkingStarted && !inThinking {
				if s.useColors {
					fmt.Fprint(s.output, colorGreen)
					thinkingStarted = true
				}
			}
			fmt.Fprint(s.output, chunk)
			buffer.Reset()
			buffer.WriteString(chunk)
		}
		
		return nil
	})

	if err != nil {
		return "", err
	}

	// Reset colors
	if s.useColors {
		fmt.Fprint(s.output, colorReset)
	}

	// If we collected content after thinking tags, render it with markdown
	if thinkingClosed && afterThinkingContent.Len() > 0 {
		finalContent := afterThinkingContent.String()
		if strings.TrimSpace(finalContent) != "" {
			if s.renderMarkdown && s.mdRenderer != nil {
				rendered, err := s.mdRenderer.Render(finalContent)
				if err == nil {
					fmt.Fprint(s.output, rendered)
				} else {
					fmt.Fprintln(s.output, finalContent)
				}
			} else {
				fmt.Fprintln(s.output, finalContent)
			}
		}
	} else {
		// No thinking tags - render everything with markdown
		if !thinkingStarted {
			response := fullResponse.String()
			s.printAssistant(response)
		} else {
			fmt.Fprintln(s.output)
		}
	}

	return fullResponse.String(), nil
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

	case "/markdown":
		s.renderMarkdown = !s.renderMarkdown
		status := "enabled"
		if !s.renderMarkdown {
			status = "disabled"
		}
		s.println(s.colorize(colorYellow, fmt.Sprintf("Markdown rendering %s.", status)))
		return false, nil

	default:
		return false, fmt.Errorf("unknown command %q. Try /help", cmd)
	}
}

func (s *Session) printWelcome() {
	s.println(s.colorize(colorCyan, fmt.Sprintf("=== Chatty v%s ===", s.version)))
	s.println(fmt.Sprintf("Model: %s | Temperature: %.1f", s.config.Model.Name, s.config.Model.Temperature))
	s.println(s.colorize(colorYellow, "Type /help for commands, /exit to quit"))
	s.println("")
}

func (s *Session) printHelp() {
	help := `Available commands:
  /help     - Show this help message
  /exit     - Exit the chat
  /reset    - Clear conversation history
  /history  - Show conversation history
  /markdown - Toggle markdown rendering`
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
	if s.renderMarkdown && s.mdRenderer != nil {
		// Render markdown
		rendered, err := s.mdRenderer.Render(text)
		if err != nil {
			// Fallback to plain text if rendering fails
			s.println(s.colorize(colorGreen, text))
			return
		}
		fmt.Fprint(s.output, rendered)
	} else {
		// Plain text mode
		s.println(s.colorize(colorGreen, text))
	}
}

func (s *Session) printWithThinkingTags(text string) {
	// Split by thinking tags and print with different colors
	thinkTagPattern := regexp.MustCompile(`(<think>|<thinking>)([\s\S]*?)(</think>|</thinking>)`)
	
	lastEnd := 0
	matches := thinkTagPattern.FindAllStringSubmatchIndex(text, -1)
	
	for _, match := range matches {
		// Print content before thinking tag (should be minimal/none)
		if match[0] > lastEnd && s.useColors {
			beforeThinking := text[lastEnd:match[0]]
			if strings.TrimSpace(beforeThinking) != "" {
				fmt.Fprint(s.output, colorGreen+beforeThinking+colorReset)
			}
		}
		
		// Print thinking content in dimmed magenta
		if s.useColors {
			fmt.Fprint(s.output, styleDim+colorMagenta+text[match[0]:match[1]]+colorReset)
		} else {
			fmt.Fprint(s.output, text[match[0]:match[1]])
		}
		
		lastEnd = match[1]
	}
	
	// Print remaining content after last thinking tag with markdown rendering
	if lastEnd < len(text) {
		finalResponse := text[lastEnd:]
		if strings.TrimSpace(finalResponse) != "" {
			// Render the final response with markdown if enabled
			if s.renderMarkdown && s.mdRenderer != nil {
				rendered, err := s.mdRenderer.Render(finalResponse)
				if err != nil {
					// Fallback to plain text
					if s.useColors {
						fmt.Fprintln(s.output, colorGreen+finalResponse+colorReset)
					} else {
						fmt.Fprintln(s.output, finalResponse)
					}
				} else {
					fmt.Fprint(s.output, rendered)
				}
			} else {
				// Plain text mode
				if s.useColors {
					fmt.Fprintln(s.output, colorGreen+finalResponse+colorReset)
				} else {
					fmt.Fprintln(s.output, finalResponse)
				}
			}
		}
	} else {
		fmt.Fprintln(s.output)
	}
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
