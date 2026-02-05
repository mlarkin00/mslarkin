package main

import (
	"fmt"
	"io" // Added for io.Writer in Render interface

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/mslarkin/models/model-chat/internal/ai"
	"github.com/mslarkin/models/model-chat/internal/ui"
	"github.com/sashabaranov/go-openai"
)

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Static files
	e.Static("/static", "static")

	// AI Client
	aiClient := ai.NewClient()

	// Routes
	e.GET("/", func(c echo.Context) error {
		// Default to first model
		defaultModel := ai.SupportedModels[0].ID
		page := ui.Layout("Vertex AI Chat", ui.App(ai.SupportedModels, defaultModel))
		c.Response().Header().Set("Content-Type", "text/html")
		return page.Render(c.Response().Writer)
	})

	e.POST("/chat", func(c echo.Context) error {
		msg := c.FormValue("message")
		modelID := c.FormValue("model_id")

		if msg == "" {
			return nil
		}

		// Render User Message immediately (though HTMX handles the request/response cycle, usually we want to append BOTH user message and AI response,
		// but standard HTMX swap replaces the target with the response.
		// A common pattern is to return: UserMessage + generic "Loading..." indicator, then swap that out, OR just return UserMessage + AIMessage.
		// For simplicity in this non-streaming version, we'll return both.

		// In a real app we might use OOB swaps or streaming.
		// Let's just return the User message and the AI message concatenated.

		userMsgNode := ui.Message(true, msg)

		// Call AI
		// Note: existing history is not passed in this simple demo version, making it single-turn effectively.
		// To support multi-turn, we'd need to store state or pass it back and forth.
		// For this "simple app", let's start with single turn or just passed context if easy.
		// We'll just send the current message for now to prove connectivity.

		respContent, err := aiClient.Chat(c.Request().Context(), modelID, []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: msg},
		})

		var aiMsgNode interface {
			Render(w io.Writer) error
		}
		// gomponents.Node

		if err != nil {
			aiMsgNode = ui.Message(false, fmt.Sprintf("Error: %v", err))
		} else {
			aiMsgNode = ui.Message(false, respContent)
		}

		c.Response().Header().Set("Content-Type", "text/html")
		userMsgNode.Render(c.Response().Writer)
		aiMsgNode.Render(c.Response().Writer)

		// Wait, gomponents Node Render takes io.Writer.
		// Echo Response Writer satisfies io.Writer.

		return nil
	})

	// Fix strict typing for gomponents in the handler above
	// Refactoring cleanup in next tool call if needed, but logic stands: write user msg, then ai msg.

	e.Logger.Fatal(e.Start(":8080"))
}
