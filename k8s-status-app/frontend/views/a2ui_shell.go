package views

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
    "k8s-status-frontend/components"
)

func A2UIShell(prompt string) Node {
    return components.Layout("GKE Status - A2UI",
        Div(ID("a2ui-root"), Class("min-h-screen p-4"),
            // Hidden input to store initial prompt if needed,
            // or we script it directly.
            Script(Rawf(`
                import { Client } from '/static/js/a2ui-core.js';
                import '/static/js/a2ui-lit.js';

                document.addEventListener('DOMContentLoaded', async () => {
                    const root = document.getElementById('a2ui-root');
                    const initialPrompt = "%s";

                    if (initialPrompt) {
                        await sendPrompt(initialPrompt);
                    }
                });

                async function sendPrompt(text) {
                     const root = document.getElementById('a2ui-root');
                     // Clear previous? Or append? For dashboard view, likely replace.
                     // But A2UI usually is a stream of components.
                     // For this demo, let's append a new interaction container.

                     // Placeholder for loading
                     const loading = document.createElement('div');
                     loading.className = 'loading loading-spinner';
                     root.appendChild(loading);

                     try {
                        const response = await fetch('/chat/proxy', {
                            method: 'POST',
                            headers: {'Content-Type': 'application/json'},
                            body: JSON.stringify({message: text, session: localStorage.getItem('chat_session') || 'default'})
                        });

                        const reader = response.body.getReader();
                        const decoder = new TextDecoder();
                        let buffer = '';

                        // We need to parse the SSE stream and extract the JSON.
                        // The Agent returns chunks. This simple demo assumes the Agent returns
                        // valid JSON chunks or we accumulate.
                        // Ideally, we'd use the official A2UI Client Transport.
                        // Since we are hacking the transport:

                        while (true) {
                            const {done, value} = await reader.read();
                            if (done) break;
                            const chunk = decoder.decode(value);
                            // Very naive SSE parsing for demo
                             const lines = chunk.split('\n\n');
                            for (const line of lines) {
                                if (line.startsWith('data: ')) {
                                    const dataStr = line.substring(6);
                                    try {
                                        const event = JSON.parse(dataStr);
                                        // The agent might return text parts that form the JSON.
                                        // OR if using ADK tools, it might return the result structure directly.
                                        // Given the prompt instruction "output JSON", we expect the text parts to BE json.

                                        if (event.parts) {
                                            event.parts.forEach(p => buffer += p.text || '');
                                        }
                                    } catch(e) {}
                                }
                            }
                        }

                        // Remove loading
                        root.removeChild(loading);

                        // Try to parse the full buffer as A2UI JSON
                        try {
                            const a2uiData = JSON.parse(buffer);

                            // The A2UI Lit component is <a2ui-root>.
                            // It usually expects a client object or specific properties.
                            // Looking at the source, it iterates over 'components' and renders them.
                            // However, directly assigning .data or .components might not be enough if it needs a 'Client' model.
                            // But let's try setting properties directly if it exposes them, or creating a wrapper.
                            // Actually, <a2ui-root> takes components property in some versions, or component (singular root).
                            // Let's assume we pass the root component to it.

                            const renderer = document.createElement('a2ui-root');

                            // Initialize Client/Model if needed.
                            const client = new Client();
                            // If the JSON is a list of components:
                            if (Array.isArray(a2uiData)) {
                                // wrap in container
                                renderer.components = a2uiData;
                            } else {
                                // Single root
                                renderer.components = [a2uiData];
                            }

                            // It seems 'components' setter on a2ui-root triggers update.
                            root.appendChild(renderer);

                        } catch (e) {
                            console.error("Failed to parse A2UI JSON", e);
                            const errDiv = document.createElement('div');
                            errDiv.innerText = "Error rendering response: " + buffer;
                            root.appendChild(errDiv);
                        }

                     } catch (e) {
                         console.error(e);
                     }
                }

                // Expose for buttons
                window.sendPrompt = sendPrompt;
            `, prompt)),
        ),
    )
}
