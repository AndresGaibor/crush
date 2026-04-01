package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
)

// AskUserHandler responde a una pregunta del agente.
type AskUserHandler func(context.Context, string) (string, error)

type askUserInput struct {
	Question string `json:"question" description:"Pregunta para el usuario"`
}

// BuildAskUserTool construye la herramienta AskUser.
func BuildAskUserTool(handler AskUserHandler) fantasy.AgentTool {
	if handler == nil {
		handler = defaultAskUserHandler
	}

	return fantasy.NewAgentTool(
		"ask_user",
		`Ask the user for clarification or confirmation and wait for a response.`,
		func(ctx context.Context, input askUserInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = call

			question := strings.TrimSpace(input.Question)
			if question == "" {
				return fantasy.NewTextErrorResponse("question is required"), nil
			}

			response, err := handler(ctx, question)
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("ask user failed: %s", err.Error())), nil
			}

			return fantasy.NewTextResponse(strings.TrimSpace(response)), nil
		},
	)
}

func defaultAskUserHandler(ctx context.Context, question string) (string, error) {
	_ = ctx

	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if stat.Mode()&os.ModeCharDevice == 0 {
		return "", fmt.Errorf("interactive input is not available")
	}

	_, _ = fmt.Fprint(os.Stdout, question)
	if !strings.HasSuffix(question, "\n") {
		_, _ = fmt.Fprint(os.Stdout, "\n")
	}
	_, _ = fmt.Fprint(os.Stdout, "> ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(answer), nil
}
