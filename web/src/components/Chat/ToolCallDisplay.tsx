import { useState } from "react";
import type { ToolCall } from "../../lib/types";

interface ToolCallDisplayProps {
	toolCall: ToolCall;
}

export function ToolCallDisplay({ toolCall }: ToolCallDisplayProps) {
	const [expanded, setExpanded] = useState(false);

	const statusColors = {
		pending: "text-yellow-400 bg-yellow-400/10",
		completed: "text-green-400 bg-green-400/10",
		error: "text-red-400 bg-red-400/10",
	};

	const statusIcons = {
		pending: "⏳",
		completed: "✓",
		error: "✗",
	};

	return (
		<div className="bg-gray-800/50 rounded-lg overflow-hidden border border-gray-600/50">
			<button
				type="button"
				onClick={() => setExpanded(!expanded)}
				className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-gray-700/50 transition-colors"
			>
				<span
					className={`text-xs px-2 py-0.5 rounded ${statusColors[toolCall.status]}`}
				>
					{statusIcons[toolCall.status]}
				</span>
				<span className="font-mono text-sm text-blue-300 flex-1 truncate">
					{toolCall.name}
				</span>
				<span className="text-gray-500 text-sm">{expanded ? "▼" : "▶"}</span>
			</button>

			{expanded && (
				<div className="px-3 pb-3 space-y-2">
					{toolCall.input && (
						<div>
							<div className="text-xs text-gray-500 mb-1">Input:</div>
							<pre className="text-xs bg-gray-900 p-2 rounded overflow-x-auto">
								<code className="text-gray-300">
									{JSON.stringify(toolCall.input, null, 2)}
								</code>
							</pre>
						</div>
					)}

					{toolCall.output && (
						<div>
							<div className="text-xs text-gray-500 mb-1">Output:</div>
							<pre className="text-xs bg-gray-900 p-2 rounded overflow-x-auto max-h-48 overflow-y-auto">
								<code className="text-gray-300">{toolCall.output}</code>
							</pre>
						</div>
					)}
				</div>
			)}
		</div>
	);
}
