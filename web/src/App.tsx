import { useState } from "react";
import { Auth } from "./components/Auth";
import { ChatPanel } from "./components/Chat";
import { FileBrowser } from "./components/File/FileBrowser";
import { GitDiffViewer } from "./components/File/GitDiffViewer";
import { SessionSidebar } from "./components/Session";
import { PermissionDialog, QuestionDialog } from "./components/ui";
import { useFileStore } from "./lib/fileStore";
import { useWsStore } from "./lib/wsStore";

function App() {
	const connectionState = useWsStore((s) => s.connectionState);
	const error = useWsStore((s) => s.error);
	const clearError = useWsStore((s) => s.clearError);
	const currentSessionId = useWsStore((s) => s.currentSessionId);
	const [sidebarOpen, setSidebarOpen] = useState(false);

	const showFileBrowser = useFileStore((s) => s.showFileBrowser);
	const showGitDiff = useFileStore((s) => s.showGitDiff);
	const setShowFileBrowser = useFileStore((s) => s.setShowFileBrowser);
	const setShowGitDiff = useFileStore((s) => s.setShowGitDiff);

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
					<div className="flex-1" />
					{/* File Browser button */}
					<button
						type="button"
						onClick={() => setShowFileBrowser(true)}
						className="p-2 text-gray-400 hover:text-white rounded-lg hover:bg-gray-700"
						title="File Browser"
					>
						<svg
							className="w-5 h-5"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<title>Files</title>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
							/>
						</svg>
					</button>
					{/* Git Diff button */}
					<button
						type="button"
						onClick={() => setShowGitDiff(true)}
						className="p-2 text-gray-400 hover:text-white rounded-lg hover:bg-gray-700"
						title="Git Changes"
					>
						<svg
							className="w-5 h-5"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<title>Git</title>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
							/>
						</svg>
					</button>
				</header>

				{/* Chat area */}
				<main className="flex-1 min-h-0">
					<ChatPanel />
				</main>
			</div>

			{/* Dialogs */}
			<PermissionDialog />
			<QuestionDialog />

			{/* File Browser Panel */}
			{showFileBrowser && (
				<div className="fixed inset-0 z-40 md:inset-auto md:right-0 md:top-0 md:bottom-0 md:w-96">
					<button
						type="button"
						className="absolute inset-0 bg-black/50 md:hidden cursor-default"
						onClick={() => setShowFileBrowser(false)}
						onKeyDown={(e) => e.key === "Escape" && setShowFileBrowser(false)}
						aria-label="Close file browser"
					/>
					<div className="absolute inset-y-0 right-0 w-full max-w-sm md:max-w-none md:w-96 bg-gray-900 shadow-xl">
						<FileBrowser />
					</div>
				</div>
			)}

			{/* Git Diff Panel */}
			{showGitDiff && (
				<div className="fixed inset-0 z-40 md:inset-auto md:right-0 md:top-0 md:bottom-0 md:w-96">
					<button
						type="button"
						className="absolute inset-0 bg-black/50 md:hidden cursor-default"
						onClick={() => setShowGitDiff(false)}
						onKeyDown={(e) => e.key === "Escape" && setShowGitDiff(false)}
						aria-label="Close git diff viewer"
					/>
					<div className="absolute inset-y-0 right-0 w-full max-w-sm md:max-w-none md:w-96 bg-gray-900 shadow-xl">
						<GitDiffViewer />
					</div>
				</div>
			)}

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
