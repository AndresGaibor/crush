package agent

import (
	"context"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/message"
)

type SendMessageParams struct {
	SessionID string `json:"session_id,omitempty" description:"Sesión de destino; si se omite, usa la sesión actual"`
	Message   string `json:"message" description:"Mensaje a enviar"`
}

const SendMessageToolName = "send_message"

func (c *coordinator) sendMessageTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		SendMessageToolName,
		`Append a message to a target session so it can be picked up later by that session's agent.`,
		func(ctx context.Context, input SendMessageParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = call

			sessionID := strings.TrimSpace(input.SessionID)
			if sessionID == "" {
				sessionID = tools.GetSessionFromContext(ctx)
			}
			if sessionID == "" {
				return fantasy.NewTextErrorResponse("session_id is required"), nil
			}

			msgText := strings.TrimSpace(input.Message)
			if msgText == "" {
				return fantasy.NewTextErrorResponse("message is required"), nil
			}

			if _, err := c.sessions.Get(ctx, sessionID); err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to find target session: %s", err.Error())), nil
			}

			msg, err := c.messages.Create(ctx, sessionID, message.CreateMessageParams{
				Role:  message.User,
				Parts: []message.ContentPart{message.TextContent{Text: msgText}},
			})
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to store message: %s", err.Error())), nil
			}

			return fantasy.NewTextResponse(fmt.Sprintf("Message added to session %s (%s).", sessionID, msg.ID)), nil
		},
	)
}
