Here's a simple Go application that hosts a Google ADK UI for chatting with different models. This implementation uses the Google Cloud ADK framework with a basic web interface:

```go
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
	models "cloud.google.com/go/vertexai/genai"
)

// Model represents a selectable AI model
type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ChatRequest represents the structure of incoming chat requests
type ChatRequest struct {
	ModelID string `json:"model_id"`
	Message string `json:"message"`
}

// ChatResponse represents the structure of chat responses
type ChatResponse struct {
	Response string `json:"response"`
}

var (
	projectID  = "mslarkin-ext"
	modelsList = []Model{
		{ID: "qwen/qwen3-next-80b-a3b-thinking-maas", Name: "Qwen 3 Next 80B"},
		{ID: "qwen/qwen3-coder-480b-a35b-instruct-maas", Name: "Qwen 3 Coder 480B"},
		{ID: "publishers/zai-org/models/glm-4.7:GLM-4.7-FP8", Name: "GLM 4.7"},
		{ID: "publishers/minimaxai/models/minimax-m2-maas", Name: "Minimax M2"},
	}
)

//go:embed index.html
var indexHTML string

func main() {
	// Set up Gin router
	router := gin.Default()

	// Serve static files
	router.GET("/", serveUI)
	router.GET("/models", getModels)
	router.POST("/chat", chatWithModel)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	log.Fatal(router.Run(":" + port))
}

// serveUI serves the main HTML page
func serveUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(indexHTML))
}

// getModels returns the list of available models
func getModels(c *gin.Context) {
	c.JSON(http.StatusOK, modelsList)
}

// chatWithModel handles chat requests to the selected model
func chatWithModel(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Find selected model
	var selectedModel *Model
	for _, model := range modelsList {
		if model.ID == req.ModelID {
			selectedModel = &model
			break
		}
	}
	if selectedModel == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	// Initialize Vertex AI client
	ctx := context.Background()
	client, err := models.NewClient(ctx, projectID, "us-west1", option.WithQuotaProject(projectID))
	if err != nil {
		// Fallback to us-central1
		client, err = models.NewClient(ctx, projectID, "us-central1", option.WithQuotaProject(projectID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize AI client"})
			return
		}
	}
	defer client.Close()

	// Initialize model
	model := client.GenerativeModel(selectedModel.ID)

	// Generate response
	resp, err := model.GenerateContent(ctx, models.Text(req.Message))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Model error: %v", err)})
		return
	}

	// Extract response text
	var responseText string
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		if text, ok := resp.Candidates[0].Content.Parts[0].(models.Text); ok {
			responseText = string(text)
		}
	}

	c.JSON(http.StatusOK, ChatResponse{Response: responseText})
}
```

Create an `index.html` file with the following content:

```html
<!DOCTYPE html>
<html>
<head>
    <title>Model Chat Interface</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .container { display: flex; flex-direction: column; gap: 20px; }
        select, textarea, button { padding: 10px; font-size: 16px; }
        textarea { min-height: 100px; }
        #response { background-color: #f5f5f5; padding: 15px; border-radius: 5px; }
        .hidden { display: none; }
    </style>
</head>
<body>
    <div class="container">
        <h1>AI Model Chat</h1>

        <div>
            <label for="modelSelect">Select Model:</label>
            <select id="modelSelect">
                <option value="">-- Select a Model --</option>
            </select>
        </div>

        <div id="chatContainer" class="hidden">
            <div>
                <label for="userInput">Your Message:</label>
                <textarea id="userInput" placeholder="Type your message here..."></textarea>
            </div>
            <button id="sendButton">Send Message</button>
            <div id="response"></div>
        </div>
    </div>

    <script>
        // Fetch and populate models
        async function loadModels() {
            const response = await fetch('/models');
            const models = await response.json();
            const select = document.getElementById('modelSelect');

            models.forEach(model => {
                const option = document.createElement('option');
                option.value = model.id;
                option.textContent = model.name;
                select.appendChild(option);
            });

            select.addEventListener('change', toggleChat);
        }

        // Toggle chat interface based on model selection
        function toggleChat() {
            const chatContainer = document.getElementById('chatContainer');
            chatContainer.classList.toggle('hidden', !this.value);
            document.getElementById('response').textContent = '';
            document.getElementById('userInput').value = '';
        }

        // Handle chat submission
        async function sendMessage() {
            const modelId = document.getElementById('modelSelect').value;
            const message = document.getElementById('userInput').value;
            const responseDiv = document.getElementById('response');

            if (!modelId || !message.trim()) return;

            responseDiv.textContent = 'Thinking...';

            try {
                const response = await fetch('/chat', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ model_id: modelId, message: message })
                });

                const data = await response.json();
                responseDiv.textContent = data.response || 'No response received';
            } catch (error) {
                responseDiv.textContent = `Error: ${error.message}`;
            }
        }

        // Initialize
        document.addEventListener('DOMContentLoaded', () => {
            loadModels();
            document.getElementById('modelSelect').addEventListener('change', toggleChat);
            document.getElementById('sendButton').addEventListener('click', sendMessage);
            document.getElementById('userInput').addEventListener('keypress', (e) => {
                if (e.key === 'Enter' && e.ctrlKey) sendMessage();
            });
        });
    </script>
</body>
</html>
```

Create a `go.mod` file:

```go
module model-chat-app

go 1.21

require (
    cloud.google.com/go/vertexai v1.0.0
    github.com/gin-gonic/gin v1.9.1
    google.golang.org/api v0.149.0
)

require (
    cloud.google.com/go v0.110.10 // indirect
    github.com/bytedance/sonic v1.10.0 // indirect
    github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad87f6980 // indirect
    github.com/gin-contrib/sse v0.1.0 // indirect
    github.com/go-playground/locales v0.14.1 // indirect
    github.com/go-playground/universal-translator v0.18.1 // indirect
    github.com/go-playground/validator/v10 v10.14.0 // indirect
    github.com/goccy/go-json v0.10.2 // indirect
    github.com/google/s2a-go v0.1.7 // indirect
    github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
    github.com/json-iterator/go v1.1.12 // indirect
    github.com/klauspost/cpuid/v2 v2.2.4 // indirect
    github.com/leodido/go-urn v1.2.4 // indirect
    github.com/mattn/go-isatty v0.0.19 // indirect
    github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
    github.com/modern-go/reflect2 v1.0.2 // indirect
    github.com/pelletier/go-toml/v2 v2.0.8 // indirect
    github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
    github.com/ugorji/go/codec v1.2.11 // indirect
    golang.org/x/arch v0.3.0 // indirect
    golang.org/x/crypto v0.12.0 // indirect
    golang.org/x/net v0.14.0 // indirect
    golang.org/x/oauth2 v0.11.0 // indirect
    golang.org/x/sync v0.3.0 // indirect
    golang.org/x/sys v0.11.0 // indirect
    golang.org/x/text v0.12.0 // indirect
    golang.org/x/time v0.3.0 // indirect
    google.golang.org/appengine v1.6.7 // indirect
    google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
    google.golang.org/grpc v1.57.0 // indirect
    google.golang.org/protobuf v1.31.0 // indirect
    gopkg.in/yaml.v3 v3.0.1 // indirect
)
```

To run this application:

1. Set up authentication:
```bash
export GOOGLE_APPLICATION_CREDENTIALS="path/to/your/service-account-key.json"
```

2. Run the application:
```bash
go run main.go
```

Key features of this implementation:
- Model selection dropdown with all specified models
- Simple chat interface with message input and response display
- Automatic region fallback (us-west1 → us-central1)
- Responsive UI with keyboard shortcuts (Ctrl+Enter to send)
- Error handling for invalid models and API errors
- Embedded HTML for single-file deployment
- JSON API endpoints for models and chat

The application will be available at http://localhost:8080. When deployed to Google Cloud Run or similar, it will automatically use the PORT environment variable.

Note: You'll need to ensure your Google Cloud project has the Vertex AI API enabled and your service account has appropriate permissions to access the specified models.
