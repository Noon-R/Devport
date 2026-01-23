import { useEffect } from "react";
import { type FileInfo, useFileStore } from "../../lib/fileStore";

export function FileBrowser() {
	const {
		currentPath,
		files,
		selectedFile,
		fileContent,
		isLoading,
		error,
		loadDirectory,
		loadFile,
		clearSelection,
		clearError,
		setShowFileBrowser,
	} = useFileStore();

	useEffect(() => {
		loadDirectory("/");
	}, [loadDirectory]);

	const handleFileClick = (file: FileInfo) => {
		if (file.is_dir) {
			loadDirectory(file.path);
		} else {
			loadFile(file.path);
		}
	};

	const handleBack = () => {
		if (selectedFile) {
			clearSelection();
		} else if (currentPath !== "/") {
			const parentPath =
				currentPath.substring(0, currentPath.lastIndexOf("/")) || "/";
			loadDirectory(parentPath);
		}
	};

	const formatSize = (size: number): string => {
		if (size < 1024) return `${size} B`;
		if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
		return `${(size / (1024 * 1024)).toFixed(1)} MB`;
	};

	return (
		<div className="flex flex-col h-full bg-gray-900">
			{/* Header */}
			<div className="flex items-center justify-between p-3 border-b border-gray-700">
				<div className="flex items-center gap-2">
					<button
						type="button"
						onClick={handleBack}
						disabled={currentPath === "/" && !selectedFile}
						className="p-1.5 rounded hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
						aria-label="Go back"
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
								d="M15 19l-7-7 7-7"
							/>
						</svg>
					</button>
					<span className="text-sm text-gray-300 truncate max-w-[200px]">
						{selectedFile ? selectedFile.name : currentPath}
					</span>
				</div>
				<button
					type="button"
					onClick={() => setShowFileBrowser(false)}
					className="p-1.5 rounded hover:bg-gray-700"
					aria-label="Close file browser"
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
				) : selectedFile && fileContent !== null ? (
					<FileViewer content={fileContent} fileName={selectedFile.name} />
				) : (
					<FileList
						files={files}
						onFileClick={handleFileClick}
						formatSize={formatSize}
					/>
				)}
			</div>
		</div>
	);
}

interface FileListProps {
	files: FileInfo[];
	onFileClick: (file: FileInfo) => void;
	formatSize: (size: number) => string;
}

function FileList({ files, onFileClick, formatSize }: FileListProps) {
	if (files.length === 0) {
		return (
			<div className="flex items-center justify-center h-32 text-gray-500">
				Empty directory
			</div>
		);
	}

	return (
		<ul className="divide-y divide-gray-800">
			{files.map((file) => (
				<li key={file.path}>
					<button
						type="button"
						onClick={() => onFileClick(file)}
						className="w-full flex items-center gap-3 p-3 hover:bg-gray-800 text-left"
					>
						{file.is_dir ? (
							<svg
								className="w-5 h-5 text-yellow-500 shrink-0"
								fill="currentColor"
								viewBox="0 0 20 20"
								aria-hidden="true"
							>
								<path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
							</svg>
						) : (
							<svg
								className="w-5 h-5 text-gray-400 shrink-0"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
								/>
							</svg>
						)}
						<div className="flex-1 min-w-0">
							<div className="text-sm text-gray-200 truncate">{file.name}</div>
							{!file.is_dir && (
								<div className="text-xs text-gray-500">
									{formatSize(file.size)}
								</div>
							)}
						</div>
						{file.is_dir && (
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
						)}
					</button>
				</li>
			))}
		</ul>
	);
}

interface FileViewerProps {
	content: string;
	fileName: string;
}

function FileViewer({ content, fileName }: FileViewerProps) {
	const lines = content.split("\n");
	const extension = fileName.split(".").pop()?.toLowerCase() || "";

	// Simple syntax class based on extension
	const getLineClass = () => {
		const codeExts = [
			"js",
			"ts",
			"tsx",
			"jsx",
			"go",
			"py",
			"rs",
			"java",
			"c",
			"cpp",
			"h",
			"hpp",
			"css",
			"html",
			"json",
			"xml",
			"md",
			"yaml",
			"yml",
			"toml",
		];
		if (codeExts.includes(extension)) {
			return "font-mono";
		}
		return "";
	};

	return (
		<div className={`p-3 ${getLineClass()}`}>
			<pre className="text-sm text-gray-300 whitespace-pre-wrap break-words overflow-x-auto">
				{lines.map((line, i) => (
					<div key={`line-${i}-${line.slice(0, 20)}`} className="flex">
						<span className="select-none text-gray-600 w-10 shrink-0 text-right pr-3">
							{i + 1}
						</span>
						<span className="flex-1">{line || " "}</span>
					</div>
				))}
			</pre>
		</div>
	);
}
