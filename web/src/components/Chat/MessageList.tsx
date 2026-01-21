import { useEffect, useRef } from "react";
import ReactMarkdown from "react-markdown";
import type { Message } from "../../lib/types";
import { ToolCallDisplay } from "./ToolCallDisplay";

interface MessageListProps {
	messages: Message[];
	isGenerating: boolean;
}

export function MessageList({ messages, isGenerating }: MessageListProps) {
	const bottomRef = useRef<HTMLDivElement>(null);

	// biome-ignore lint/correctness/useExhaustiveDependencies: scroll on message change
	useEffect(() => {
		bottomRef.current?.scrollIntoView({ behavior: "smooth" });
	}, [messages]);

	if (messages.length === 0) {
		return (
			<div className="flex-1 flex items-center justify-center text-gray-500">
				<div className="text-center">
					<p className="text-lg mb-2">Start a conversation</p>
					<p className="text-sm">Type your message below</p>
				</div>
			</div>
		);
	}

	return (
		<div className="flex-1 overflow-y-auto p-4 space-y-4">
			{messages.map((message) => (
				<MessageBubble key={message.id} message={message} />
			))}
			{isGenerating && (
				<div className="flex items-center gap-2 text-gray-400 pl-4">
					<div className="flex gap-1">
						<span className="w-2 h-2 bg-blue-400 rounded-full animate-bounce" />
						<span
							className="w-2 h-2 bg-blue-400 rounded-full animate-bounce"
							style={{ animationDelay: "0.1s" }}
						/>
						<span
							className="w-2 h-2 bg-blue-400 rounded-full animate-bounce"
							style={{ animationDelay: "0.2s" }}
						/>
					</div>
					<span className="text-sm">Thinking...</span>
				</div>
			)}
			<div ref={bottomRef} />
		</div>
	);
}

function MessageBubble({ message }: { message: Message }) {
	const isUser = message.role === "user";
	const isSystem = message.role === "system";

	if (isSystem) {
		return (
			<div className="flex justify-center">
				<div className="bg-gray-700/50 text-gray-400 text-sm px-4 py-2 rounded-full">
					{message.content}
				</div>
			</div>
		);
	}

	return (
		<div className={`flex ${isUser ? "justify-end" : "justify-start"}`}>
			<div
				className={`max-w-[85%] md:max-w-[70%] ${
					isUser
						? "bg-blue-600 text-white rounded-2xl rounded-br-sm"
						: "bg-gray-700 text-gray-100 rounded-2xl rounded-bl-sm"
				} px-4 py-3`}
			>
				{message.content && (
					<div className="prose prose-invert prose-sm max-w-none break-words">
						<ReactMarkdown>{message.content}</ReactMarkdown>
					</div>
				)}

				{message.toolCalls && message.toolCalls.length > 0 && (
					<div className="mt-3 space-y-2">
						{message.toolCalls.map((toolCall) => (
							<ToolCallDisplay key={toolCall.id} toolCall={toolCall} />
						))}
					</div>
				)}
			</div>
		</div>
	);
}
