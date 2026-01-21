import { useState } from "react";
import { useWsStore } from "../../lib/wsStore";

export function QuestionDialog() {
	const pendingQuestion = useWsStore((s) => s.pendingQuestion);
	const respondToQuestion = useWsStore((s) => s.respondToQuestion);
	const [customAnswer, setCustomAnswer] = useState("");
	const [showCustomInput, setShowCustomInput] = useState(false);

	if (!pendingQuestion) return null;

	const handleOptionClick = (label: string) => {
		respondToQuestion(label);
	};

	const handleCustomSubmit = () => {
		if (customAnswer.trim()) {
			respondToQuestion(customAnswer.trim());
			setCustomAnswer("");
			setShowCustomInput(false);
		}
	};

	return (
		<div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
			<div className="bg-gray-800 rounded-xl shadow-2xl max-w-md w-full p-6 border border-gray-700">
				<div className="flex items-center gap-3 mb-4">
					<div className="w-10 h-10 bg-blue-500/20 rounded-full flex items-center justify-center">
						<svg
							className="w-6 h-6 text-blue-400"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<title>Question</title>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
					</div>
					<h3 className="text-lg font-semibold text-white">Question</h3>
				</div>

				<p className="text-gray-300 mb-4">{pendingQuestion.question}</p>

				{!showCustomInput ? (
					<div className="space-y-2 mb-4">
						{pendingQuestion.options.map((option) => (
							<button
								key={option.label}
								type="button"
								onClick={() => handleOptionClick(option.label)}
								className="w-full text-left p-3 bg-gray-700 hover:bg-gray-600 rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
							>
								<div className="text-white font-medium">{option.label}</div>
								{option.description && (
									<div className="text-sm text-gray-400 mt-1">
										{option.description}
									</div>
								)}
							</button>
						))}
						<button
							type="button"
							onClick={() => setShowCustomInput(true)}
							className="w-full text-left p-3 bg-gray-700/50 hover:bg-gray-600/50 rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 border border-dashed border-gray-600"
						>
							<div className="text-gray-400">+ Custom answer</div>
						</button>
					</div>
				) : (
					<div className="space-y-3 mb-4">
						<textarea
							value={customAnswer}
							onChange={(e) => setCustomAnswer(e.target.value)}
							placeholder="Type your answer..."
							rows={3}
							className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
						/>
						<div className="flex gap-2">
							<button
								type="button"
								onClick={() => {
									setShowCustomInput(false);
									setCustomAnswer("");
								}}
								className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors"
							>
								Back
							</button>
							<button
								type="button"
								onClick={handleCustomSubmit}
								disabled={!customAnswer.trim()}
								className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white rounded-lg transition-colors"
							>
								Submit
							</button>
						</div>
					</div>
				)}
			</div>
		</div>
	);
}
