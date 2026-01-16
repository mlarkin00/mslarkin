FROM ollama/ollama:latest

ARG MODEL_NAME=gpt-oss:20b
# gpt-oss:20b, qwen3:4b, qwen3:0.6b
ENV OLLAMA_HOST=0.0.0.0:8000

# The build process needs to start the Ollama server in the background,
# wait for it to be ready, pull the model, and then stop the server.
# We combine this into a single RUN command to keep image layers clean.
RUN ollama serve & \
  sleep 5 && \
  ollama pull ${MODEL_NAME} && \
  pkill ollama

# The base image already has the correct ENTRYPOINT to start 'ollama serve',
# so no need to redefine it unless you have custom startup needs.

EXPOSE 8000

# CMD ["ollama", "serve"]
