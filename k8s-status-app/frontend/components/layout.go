package components

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func Layout(title string, body Node) Node {
	return HTML(
		Head(
			TitleEl(Text(title)),
			Script(Attr("src", "https://unpkg.com/htmx.org@1.9.10")),
			Link(Attr("rel", "stylesheet"), Attr("href", "https://cdn.jsdelivr.net/npm/daisyui@4.6.0/dist/full.min.css")),
			Script(Attr("src", "https://cdn.tailwindcss.com")),
            // Import A2UI as ES module
            Script(Type("module"), Raw(`
                import { Client } from '`+AppLink("/static/js/a2ui-core.js")+`';
                import '`+AppLink("/static/js/a2ui-lit.js")+`';
                // Expose Client if needed globally or just ensure registration
            `)),
			Script(Raw(`
                async function sendMessage(event) {
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

                    try {
                        const response = await fetch('`+AppLink("/chat/proxy")+`', {
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
            `)),
		),
		Body(
			Class("bg-gray-100 min-h-screen"),
			Navbar(),
			Container(
				body,
			),
			ChatWidget(),
		),
	)
}

func Container(children ...Node) Node {
	return Div(Class("container mx-auto p-4"), Group(children))
}

func Navbar() Node {
	return Div(Class("navbar bg-base-100 shadow-lg mb-4"),
		Div(Class("flex-1"),
			A(Class("btn btn-ghost text-xl"), Text("GKE Status")),
		),
	)
}

func ChatWidget() Node {
	return Div(Class("fixed bottom-4 right-4 w-96 bg-white shadow-xl rounded-lg border border-gray-200 overflow-hidden z-50 flex flex-col"),
		Div(Class("bg-blue-600 text-white p-3 font-bold flex justify-between items-center"),
			Text("AI Assistant"),
		),
		Div(Class("h-96 overflow-y-auto p-3 bg-gray-50 flex flex-col gap-2"),
			ID("chat-messages"),
		),
		FormEl(
			Class("p-3 border-t flex"),
			Attr("onsubmit", "sendMessage(event)"),
			Input(Type("text"), Name("message"), Class("input input-bordered w-full mr-2"), Placeholder("Ask about GKE...")),
			Button(Class("btn btn-primary"), Text("Send")),
		),
	)
}
