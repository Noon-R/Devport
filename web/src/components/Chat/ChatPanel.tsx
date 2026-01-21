import { useWsStore } from "../../lib/wsStore";
import { InputArea } from "./InputArea";
import { MessageList } from "./MessageList";

export function ChatPanel() {
	const messages = useWsStore((s) => s.messages);
	const isGenerating = useWsStore((s) => s.isGenerating);
	const currentSessionId = useWsStore((s) => s.currentSessionId);
	const sendMessage = useWsStore((s) => s.sendMessage);
	const interrupt = useWsStore((s) => s.interrupt);

	return (
		<div className="flex flex-col h-full bg-gray-900">
			<MessageList messages={messages} isGenerating={isGenerating} />
			<InputArea
				onSend={sendMessage}
				onInterrupt={interrupt}
				isGenerating={isGenerating}
				disabled={!currentSessionId}
			/>
		</div>
	);
}
