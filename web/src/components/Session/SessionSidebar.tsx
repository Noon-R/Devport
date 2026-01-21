import { useEffect, useState } from "react";
import type { Session } from "../../lib/types";
import { useWsStore } from "../../lib/wsStore";

interface SessionSidebarProps {
	isOpen: boolean;
	onClose: () => void;
}

export function SessionSidebar({ isOpen, onClose }: SessionSidebarProps) {
	const sessions = useWsStore((s) => s.sessions);
	const currentSessionId = useWsStore((s) => s.currentSessionId);
	const loadSessions = useWsStore((s) => s.loadSessions);
	const createSession = useWsStore((s) => s.createSession);
	const attachSession = useWsStore((s) => s.attachSession);
	const disconnect = useWsStore((s) => s.disconnect);
	const [isCreating, setIsCreating] = useState(false);

	useEffect(() => {
		loadSessions();
	}, [loadSessions]);

	const handleCreateSession = async () => {
		setIsCreating(true);
		try {
			const session = await createSession();
			await attachSession(session.id);
			onClose();
		} finally {
			setIsCreating(false);
		}
	};

	const handleSelectSession = async (session: Session) => {
		await attachSession(session.id);
		onClose();
	};

	return (
		<>
			{/* Backdrop */}
			{isOpen && (
				<button
					type="button"
					className="fixed inset-0 bg-black/50 z-40 md:hidden"
					onClick={onClose}
					onKeyDown={(e) => e.key === "Escape" && onClose()}
					aria-label="Close sidebar"
				/>
			)}

			{/* Sidebar */}
			<div
				className={`fixed inset-y-0 left-0 w-72 bg-gray-800 border-r border-gray-700 z-50 transform transition-transform duration-200 ease-out ${
					isOpen ? "translate-x-0" : "-translate-x-full"
				} md:translate-x-0 md:static md:z-0`}
			>
				<div className="flex flex-col h-full">
					{/* Header */}
					<div className="p-4 border-b border-gray-700">
						<div className="flex items-center justify-between mb-4">
							<h2 className="text-lg font-semibold text-white">Sessions</h2>
							<button
								type="button"
								onClick={onClose}
								className="md:hidden p-2 text-gray-400 hover:text-white"
							>
								<svg
									className="w-5 h-5"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<title>Close</title>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M6 18L18 6M6 6l12 12"
									/>
								</svg>
							</button>
						</div>
						<button
							type="button"
							onClick={handleCreateSession}
							disabled={isCreating}
							className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 text-white rounded-lg transition-colors"
						>
							<svg
								className="w-5 h-5"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<title>New session</title>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M12 4v16m8-8H4"
								/>
							</svg>
							{isCreating ? "Creating..." : "New Session"}
						</button>
					</div>

					{/* Session list */}
					<div className="flex-1 overflow-y-auto p-2">
						{sessions.length === 0 ? (
							<p className="text-gray-500 text-center py-8 text-sm">
								No sessions yet
							</p>
						) : (
							<div className="space-y-1">
								{sessions.map((session) => (
									<button
										key={session.id}
										type="button"
										onClick={() => handleSelectSession(session)}
										className={`w-full text-left p-3 rounded-lg transition-colors ${
											session.id === currentSessionId
												? "bg-blue-600/20 text-blue-300 border border-blue-500/50"
												: "text-gray-300 hover:bg-gray-700"
										}`}
									>
										<div className="font-medium truncate">{session.title}</div>
										<div className="text-xs text-gray-500 mt-1">
											{formatDate(session.updatedAt)}
										</div>
									</button>
								))}
							</div>
						)}
					</div>

					{/* Footer */}
					<div className="p-4 border-t border-gray-700">
						<button
							type="button"
							onClick={disconnect}
							className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg transition-colors"
						>
							<svg
								className="w-5 h-5"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<title>Disconnect</title>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
								/>
							</svg>
							Disconnect
						</button>
					</div>
				</div>
			</div>
		</>
	);
}

function formatDate(dateString: string): string {
	const date = new Date(dateString);
	const now = new Date();
	const diff = now.getTime() - date.getTime();

	if (diff < 60000) return "Just now";
	if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
	if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
	return date.toLocaleDateString();
}
