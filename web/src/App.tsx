import { useState } from "react";
import { Auth } from "./components/Auth";
import { ChatPanel } from "./components/Chat";
import { SessionSidebar } from "./components/Session";
import { PermissionDialog, QuestionDialog } from "./components/ui";
import { useWsStore } from "./lib/wsStore";

function App() {
	const connectionState = useWsStore((s) => s.connectionState);
	const error = useWsStore((s) => s.error);
	const clearError = useWsStore((s) => s.clearError);
	const currentSessionId = useWsStore((s) => s.currentSessionId);
	const [sidebarOpen, setSidebarOpen] = useState(false);

	// Show auth screen if not authenticated
	if (connectionState !== "authenticated") {
		return <Auth />;
	}

	return (
		<div className="h-screen flex bg-gray-900">
			{/* Session Sidebar */}
			<SessionSidebar
				isOpen={sidebarOpen}
				onClose={() => setSidebarOpen(false)}
			/>

			{/* Main content */}
			<div className="flex-1 flex flex-col min-w-0">
				{/* Header */}
				<header className="bg-gray-800 border-b border-gray-700 px-4 py-3 flex items-center gap-3">
					<button
						type="button"
						onClick={() => setSidebarOpen(true)}
						className="md:hidden p-2 text-gray-400 hover:text-white rounded-lg hover:bg-gray-700"
					>
						<svg
							className="w-6 h-6"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<title>Menu</title>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M4 6h16M4 12h16M4 18h16"
							/>
						</svg>
					</button>
					<h1 className="text-lg font-semibold text-white">Devport</h1>
					{currentSessionId && (
						<span className="text-sm text-gray-400 truncate">
							Session: {currentSessionId.slice(0, 8)}...
						</span>
					)}
				</header>

				{/* Chat area */}
				<main className="flex-1 min-h-0">
					<ChatPanel />
				</main>
			</div>

			{/* Dialogs */}
			<PermissionDialog />
			<QuestionDialog />

			{/* Error toast */}
			{error && (
				<div className="fixed bottom-4 right-4 bg-red-900 border border-red-700 rounded-lg p-4 shadow-xl max-w-sm z-50">
					<div className="flex items-start gap-3">
						<div className="flex-1">
							<p className="text-red-200 text-sm">{error}</p>
						</div>
						<button
							type="button"
							onClick={clearError}
							className="text-red-400 hover:text-red-200"
						>
							<svg
								className="w-5 h-5"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<title>Dismiss</title>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M6 18L18 6M6 6l12 12"
								/>
							</svg>
						</button>
					</div>
				</div>
			)}
		</div>
	);
}

export default App;
