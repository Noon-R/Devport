import { type FormEvent, type KeyboardEvent, useRef, useState } from "react";

interface InputAreaProps {
	onSend: (content: string) => void;
	onInterrupt: () => void;
	isGenerating: boolean;
	disabled: boolean;
}

export function InputArea({
	onSend,
	onInterrupt,
	isGenerating,
	disabled,
}: InputAreaProps) {
	const [input, setInput] = useState("");
	const textareaRef = useRef<HTMLTextAreaElement>(null);

	const handleSubmit = (e: FormEvent) => {
		e.preventDefault();
		if (input.trim() && !disabled && !isGenerating) {
			onSend(input.trim());
			setInput("");
			if (textareaRef.current) {
				textareaRef.current.style.height = "auto";
			}
		}
	};

	const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
		if (e.key === "Enter" && !e.shiftKey) {
			e.preventDefault();
			handleSubmit(e);
		}
	};

	const handleInput = () => {
		const textarea = textareaRef.current;
		if (textarea) {
			textarea.style.height = "auto";
			textarea.style.height = `${Math.min(textarea.scrollHeight, 200)}px`;
		}
	};

	return (
		<div className="border-t border-gray-700 bg-gray-800 p-4">
			<form onSubmit={handleSubmit} className="flex gap-2 items-end">
				<div className="flex-1 relative">
					<textarea
						ref={textareaRef}
						value={input}
						onChange={(e) => {
							setInput(e.target.value);
							handleInput();
						}}
						onKeyDown={handleKeyDown}
						placeholder={
							disabled ? "Select a session first" : "Type a message..."
						}
						disabled={disabled}
						rows={1}
						className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-xl text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none disabled:opacity-50 disabled:cursor-not-allowed"
					/>
				</div>

				{isGenerating ? (
					<button
						type="button"
						onClick={onInterrupt}
						className="px-4 py-3 bg-red-600 hover:bg-red-700 text-white rounded-xl transition-colors focus:outline-none focus:ring-2 focus:ring-red-500"
					>
						<svg
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
							aria-hidden="true"
						>
							<title>Stop</title>
							<rect x="4" y="4" width="12" height="12" rx="1" />
						</svg>
					</button>
				) : (
					<button
						type="submit"
						disabled={!input.trim() || disabled}
						className="px-4 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-xl transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
					>
						<svg
							className="w-5 h-5"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<title>Send</title>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"
							/>
						</svg>
					</button>
				)}
			</form>
		</div>
	);
}
