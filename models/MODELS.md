
# Goal:
Create a simple Golang application that hosts the Google ADK UI, where the user can select from a list of models, and then chat with the selected model.

## Project:
mslarkin-ext

## Regions:
- us-west1 (preferred)
- global (preferred)
- us-central1 (fallback)

## Models:
- "qwen/qwen3-coder-480b-a35b-instruct-maas", DisplayName: "Qwen 3 Coder 480B (Instruct)", IsThinking: false, Region: "us-south1"
- "qwen/qwen3-next-80b-a3b-thinking-maas", DisplayName: "Qwen 3 Next 80B (Thinking)", IsThinking: true, Region: "global"
- "google/gemini-3.0-flash", DisplayName: "Gemini 3.0 Flash", IsThinking: true, Region: "global"
- "google/gemini-3.0-pro", DisplayName: "Gemini 3.0 Pro", IsThinking: true, Region: "global"
- "deepseek-ai/deepseek-r1", DisplayName: "DeepSeek R1", IsThinking: true, Region: "global"
- "moonshot-ai/kimi-k2-thinking-maas", DisplayName: "Kimi K2 Thinking", IsThinking: true, Region: "global"

