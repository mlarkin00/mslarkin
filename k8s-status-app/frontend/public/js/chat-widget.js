import { Client } from './a2ui-core.js';
import './a2ui-lit.js';

window.sendMessage = async function(event) {
    event.preventDefault();
    const input = event.target.querySelector('input[name="message"]');
    const message = input.value;
    if (!message) return;

    const chatBox = document.getElementById('chat-messages');
    const userDiv = document.createElement('div');
    userDiv.className = "chat chat-end";
    userDiv.innerHTML = '<div class="chat-bubble chat-bubble-primary">' + message + '</div>';
    chatBox.appendChild(userDiv);
    input.value = '';

    const sessionID = localStorage.getItem('chat_session') || 'default';

    // Use configured base path or default to empty
    let basePath = window.APP_CONFIG?.basePath || '';
    if (basePath === '/') basePath = ''; // Avoid double slash if base is root
    const targetURL = `${basePath}/chat/proxy`;

    try {
        const response = await fetch(targetURL, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({message: message, session: sessionID})
        });

        const reader = response.body.getReader();
        const decoder = new TextDecoder();

        const botDiv = document.createElement('div');
        botDiv.className = "chat chat-start w-full";
        // Use a container for A2UI
        botDiv.innerHTML = '<div class="chat-bubble chat-bubble-secondary w-full p-0 bg-transparent shadow-none text-black"></div>';
        chatBox.appendChild(botDiv);
        const botContainer = botDiv.querySelector('.chat-bubble');

        // Accumulate full JSON for A2UI
        let buffer = '';

        while (true) {
            const {done, value} = await reader.read();
            if (done) break;
            const chunk = decoder.decode(value);
            const lines = chunk.split('\n\n');
            for (const line of lines) {
                if (line.startsWith('data: ')) {
                    const dataStr = line.substring(6);
                    try {
                        const event = JSON.parse(dataStr);
                        if (event.parts) {
                            event.parts.forEach(p => buffer += p.text || '');
                        }
                    } catch (e) { }
                }
            }
        }

        // Render A2UI
            try {
            const a2uiData = JSON.parse(buffer);
            const renderer = document.createElement('a2ui-root');
            if (Array.isArray(a2uiData)) {
                renderer.components = a2uiData;
            } else {
                renderer.components = [a2uiData];
            }
            botContainer.appendChild(renderer);
        } catch (e) {
                // Fallback to text
                botContainer.innerText = buffer;
        }

    } catch (e) {
        console.error("Chat failed", e);
    }
}
