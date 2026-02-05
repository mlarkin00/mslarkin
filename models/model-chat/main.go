package main

import (
	"fmt"

	"sync" // Added for parallel execution

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
		defaultModelID := ai.SupportedModels[0].ID
		page := ui.Layout("Vertex AI Chat", ui.App(ai.SupportedModels, []string{defaultModelID}))
		c.Response().Header().Set("Content-Type", "text/html")
		return page.Render(c.Response().Writer)
	})

	e.POST("/chat", func(c echo.Context) error {
		// Parse input
		msg := c.FormValue("message")
		// Parse one or more model IDs.
		// Note: Echo's c.FormValue returns only the first value. c.FormParams() returns map[string][]string.
		FormParams, err := c.FormParams()
		if err != nil {
			return fmt.Errorf("failed to parse form params: %w", err)
		}
		modelIDs := FormParams["model_id"]

		// Fallback if empty (shouldn't happen with UI defaults, but good for safety)
		if len(modelIDs) == 0 {
			// fallback to just one if passed via simpler means or default
			if single := c.FormValue("model_id"); single != "" {
				modelIDs = []string{single}
			}
		}

		if msg == "" {
			return nil
		}
		if len(modelIDs) == 0 {
			// Error or default
			return nil
		}

		// Parallel execution
		type result struct {
			modelID string
			resp    *ai.ChatResponse
			err     error
		}
		resultsCh := make(chan result, len(modelIDs))
		var wg sync.WaitGroup

		for _, mid := range modelIDs {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				resp, err := aiClient.Chat(c.Request().Context(), id, []openai.ChatCompletionMessage{
					{Role: openai.ChatMessageRoleUser, Content: msg},
				})
				resultsCh <- result{modelID: id, resp: resp, err: err}
			}(mid)
		}

		wg.Wait()
		close(resultsCh)

		// Aggregate
		responses := make(map[string]*ai.ChatResponse)
		errors := make(map[string]error)

		for res := range resultsCh {
			if res.err != nil {
				errors[res.modelID] = res.err
			} else {
				responses[res.modelID] = res.resp
			}
		}

		// Render Comparison View
		compNode := ui.ComparisonView(msg, responses, errors)

		c.Response().Header().Set("Content-Type", "text/html")
		return compNode.Render(c.Response().Writer)
	})

	// Fix strict typing for gomponents in the handler above
	// Refactoring cleanup in next tool call if needed, but logic stands: write user msg, then ai msg.

	e.Logger.Fatal(e.Start(":8080"))
}
