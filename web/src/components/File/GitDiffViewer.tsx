import { useEffect, useState } from "react";
import { type DiffFile, useFileStore } from "../../lib/fileStore";

export function GitDiffViewer() {
	const { gitDiff, isLoading, error, loadGitDiff, clearError, setShowGitDiff } =
		useFileStore();

	const [selectedFile, setSelectedFile] = useState<string | null>(null);
	const [showStaged, setShowStaged] = useState(false);

	useEffect(() => {
		loadGitDiff();
	}, [loadGitDiff]);

	const files = showStaged ? gitDiff?.staged : gitDiff?.files;
	const diff = showStaged ? gitDiff?.staged_diff : gitDiff?.diff;

	const getSelectedFileDiff = (): string => {
		if (!diff || !selectedFile) return "";

		// Parse diff to find the selected file's diff
		const lines = diff.split("\n");
		let inSelectedFile = false;
		const result: string[] = [];

		for (const line of lines) {
			if (line.startsWith("diff --git")) {
				inSelectedFile = line.includes(selectedFile);
				if (inSelectedFile) {
					result.push(line);
				}
			} else if (inSelectedFile) {
				result.push(line);
			}
		}

		return result.join("\n");
	};

	return (
		<div className="flex flex-col h-full bg-gray-900">
			{/* Header */}
			<div className="flex items-center justify-between p-3 border-b border-gray-700">
				<div className="flex items-center gap-3">
					<span className="text-sm font-medium text-gray-200">Git Changes</span>
					{gitDiff?.branch && (
						<span className="text-xs px-2 py-0.5 bg-blue-900 text-blue-200 rounded">
							{gitDiff.branch}
						</span>
					)}
				</div>
				<button
					type="button"
					onClick={() => setShowGitDiff(false)}
					className="p-1.5 rounded hover:bg-gray-700"
					aria-label="Close git diff viewer"
				>
					<svg
						className="w-5 h-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			</div>

			{/* Tabs */}
			<div className="flex border-b border-gray-700">
				<button
					type="button"
					onClick={() => {
						setShowStaged(false);
						setSelectedFile(null);
					}}
					className={`flex-1 py-2 text-sm font-medium ${!showStaged ? "text-blue-400 border-b-2 border-blue-400" : "text-gray-400"}`}
				>
					Working ({gitDiff?.files?.length || 0})
				</button>
				<button
					type="button"
					onClick={() => {
						setShowStaged(true);
						setSelectedFile(null);
					}}
					className={`flex-1 py-2 text-sm font-medium ${showStaged ? "text-green-400 border-b-2 border-green-400" : "text-gray-400"}`}
				>
					Staged ({gitDiff?.staged?.length || 0})
				</button>
			</div>

			{/* Error message */}
			{error && (
				<div className="p-3 bg-red-900/50 text-red-200 text-sm flex justify-between items-center">
					<span>{error}</span>
					<button
						type="button"
						onClick={clearError}
						className="text-red-300 hover:text-red-100"
						aria-label="Dismiss error"
					>
						<svg
							className="w-4 h-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M6 18L18 6M6 6l12 12"
							/>
						</svg>
					</button>
				</div>
			)}

			{/* Content */}
			<div className="flex-1 overflow-auto">
				{isLoading ? (
					<div className="flex items-center justify-center h-32">
						<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
					</div>
				) : !gitDiff?.has_changes && !showStaged ? (
					<div className="flex flex-col items-center justify-center h-32 text-gray-500">
						<svg
							className="w-12 h-12 mb-2"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={1.5}
								d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<span>No changes</span>
					</div>
				) : selectedFile ? (
					<DiffContent
						diff={getSelectedFileDiff()}
						onBack={() => setSelectedFile(null)}
						fileName={selectedFile}
					/>
				) : (
					<FileChangeList files={files || []} onFileClick={setSelectedFile} />
				)}
			</div>

			{/* Refresh button */}
			<div className="p-3 border-t border-gray-700">
				<button
					type="button"
					onClick={() => loadGitDiff()}
					disabled={isLoading}
					className="w-full py-2 text-sm font-medium text-gray-300 bg-gray-800 rounded hover:bg-gray-700 disabled:opacity-50"
				>
					{isLoading ? "Loading..." : "Refresh"}
				</button>
			</div>
		</div>
	);
}

interface FileChangeListProps {
	files: DiffFile[];
	onFileClick: (path: string) => void;
}

function FileChangeList({ files, onFileClick }: FileChangeListProps) {
	if (files.length === 0) {
		return (
			<div className="flex items-center justify-center h-32 text-gray-500">
				No changes
			</div>
		);
	}

	const getStatusColor = (status: string) => {
		switch (status) {
			case "added":
				return "text-green-400";
			case "deleted":
				return "text-red-400";
			case "modified":
				return "text-yellow-400";
			case "renamed":
				return "text-blue-400";
			default:
				return "text-gray-400";
		}
	};

	const getStatusIcon = (status: string) => {
		switch (status) {
			case "added":
				return "A";
			case "deleted":
				return "D";
			case "modified":
				return "M";
			case "renamed":
				return "R";
			default:
				return "?";
		}
	};

	return (
		<ul className="divide-y divide-gray-800">
			{files.map((file) => (
				<li key={file.path}>
					<button
						type="button"
						onClick={() => onFileClick(file.path)}
						className="w-full flex items-center gap-3 p-3 hover:bg-gray-800 text-left"
					>
						<span
							className={`w-5 h-5 flex items-center justify-center text-xs font-bold ${getStatusColor(file.status)}`}
						>
							{getStatusIcon(file.status)}
						</span>
						<div className="flex-1 min-w-0">
							<div className="text-sm text-gray-200 truncate">{file.path}</div>
							<div className="text-xs text-gray-500 flex gap-2">
								{file.additions > 0 && (
									<span className="text-green-400">+{file.additions}</span>
								)}
								{file.deletions > 0 && (
									<span className="text-red-400">-{file.deletions}</span>
								)}
							</div>
						</div>
						<svg
							className="w-4 h-4 text-gray-500 shrink-0"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M9 5l7 7-7 7"
							/>
						</svg>
					</button>
				</li>
			))}
		</ul>
	);
}

interface DiffContentProps {
	diff: string;
	fileName: string;
	onBack: () => void;
}

function DiffContent({ diff, fileName, onBack }: DiffContentProps) {
	const lines = diff.split("\n");

	const getLineClass = (line: string) => {
		if (line.startsWith("+") && !line.startsWith("+++")) {
			return "bg-green-900/30 text-green-300";
		}
		if (line.startsWith("-") && !line.startsWith("---")) {
			return "bg-red-900/30 text-red-300";
		}
		if (line.startsWith("@@")) {
			return "bg-blue-900/30 text-blue-300";
		}
		if (
			line.startsWith("diff") ||
			line.startsWith("index") ||
			line.startsWith("---") ||
			line.startsWith("+++")
		) {
			return "text-gray-500";
		}
		return "text-gray-300";
	};

	return (
		<div className="flex flex-col h-full">
			{/* File header */}
			<div className="flex items-center gap-2 p-3 border-b border-gray-700 bg-gray-800">
				<button
					type="button"
					onClick={onBack}
					className="p-1 rounded hover:bg-gray-700"
					aria-label="Go back"
				>
					<svg
						className="w-4 h-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M15 19l-7-7 7-7"
						/>
					</svg>
				</button>
				<span className="text-sm text-gray-200 truncate">{fileName}</span>
			</div>

			{/* Diff content */}
			<div className="flex-1 overflow-auto p-2">
				<pre className="text-xs font-mono">
					{lines.map((line, i) => (
						<div
							key={`diff-${i}-${line.slice(0, 20)}`}
							className={`px-2 ${getLineClass(line)}`}
						>
							{line || " "}
						</div>
					))}
				</pre>
			</div>
		</div>
	);
}
