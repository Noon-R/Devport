import { create } from "zustand";

// File info from server
export interface FileInfo {
	name: string;
	path: string;
	is_dir: boolean;
	size: number;
	mod_time: string;
}

// Git diff file
export interface DiffFile {
	path: string;
	status: "added" | "modified" | "deleted" | "renamed";
	additions: number;
	deletions: number;
}

// Git status response
export interface GitStatus {
	branch: string;
	is_repo: boolean;
	has_changes: boolean;
	staged: string[];
	unstaged: string[];
	untracked: string[];
}

// Git diff response
export interface GitDiff {
	branch: string;
	files: DiffFile[];
	diff: string;
	has_changes: boolean;
	staged: DiffFile[];
	staged_diff: string;
}

interface FileState {
	// Connection info
	baseUrl: string;
	token: string;

	// File browser state
	currentPath: string;
	files: FileInfo[];
	selectedFile: FileInfo | null;
	fileContent: string | null;
	isLoading: boolean;
	error: string | null;

	// Git state
	gitStatus: GitStatus | null;
	gitDiff: GitDiff | null;

	// View state
	showFileBrowser: boolean;
	showGitDiff: boolean;

	// Actions
	setConnection: (baseUrl: string, token: string) => void;
	loadDirectory: (path?: string) => Promise<void>;
	loadFile: (path: string) => Promise<void>;
	saveFile: (path: string, content: string) => Promise<void>;
	deleteFile: (path: string) => Promise<void>;
	loadGitStatus: () => Promise<void>;
	loadGitDiff: () => Promise<void>;
	setShowFileBrowser: (show: boolean) => void;
	setShowGitDiff: (show: boolean) => void;
	clearSelection: () => void;
	clearError: () => void;
}

export const useFileStore = create<FileState>((set, get) => {
	const apiFetch = async (
		endpoint: string,
		options?: RequestInit,
	): Promise<Response> => {
		const { baseUrl, token } = get();
		const url = `${baseUrl}${endpoint}`;
		const headers = {
			Authorization: `Bearer ${token}`,
			...options?.headers,
		};
		return fetch(url, { ...options, headers });
	};

	return {
		baseUrl: "",
		token: "",
		currentPath: "/",
		files: [],
		selectedFile: null,
		fileContent: null,
		isLoading: false,
		error: null,
		gitStatus: null,
		gitDiff: null,
		showFileBrowser: false,
		showGitDiff: false,

		setConnection: (baseUrl: string, token: string) => {
			// Convert WebSocket URL to HTTP URL
			const httpUrl = baseUrl
				.replace("ws://", "http://")
				.replace("wss://", "https://")
				.replace("/ws", "");
			set({ baseUrl: httpUrl, token });
		},

		loadDirectory: async (path = "/") => {
			set({ isLoading: true, error: null });
			try {
				const response = await apiFetch(`/api/fs${path}`);
				if (!response.ok) {
					throw new Error(`Failed to load directory: ${response.statusText}`);
				}
				const data = await response.json();
				set({
					currentPath: path,
					files: data.files || [],
					isLoading: false,
				});
			} catch (e) {
				set({ error: (e as Error).message, isLoading: false });
			}
		},

		loadFile: async (path: string) => {
			set({ isLoading: true, error: null });
			try {
				const response = await apiFetch(`/api/fs${path}`);
				if (!response.ok) {
					throw new Error(`Failed to load file: ${response.statusText}`);
				}
				const content = await response.text();
				const files = get().files;
				const file = files.find((f) => f.path === path);
				set({
					selectedFile: file || {
						name: path.split("/").pop() || "",
						path,
						is_dir: false,
						size: content.length,
						mod_time: "",
					},
					fileContent: content,
					isLoading: false,
				});
			} catch (e) {
				set({ error: (e as Error).message, isLoading: false });
			}
		},

		saveFile: async (path: string, content: string) => {
			set({ isLoading: true, error: null });
			try {
				const response = await apiFetch(`/api/fs${path}`, {
					method: "PUT",
					body: content,
				});
				if (!response.ok) {
					throw new Error(`Failed to save file: ${response.statusText}`);
				}
				set({ fileContent: content, isLoading: false });
				// Reload directory to update file list
				const dirPath = path.substring(0, path.lastIndexOf("/")) || "/";
				get().loadDirectory(dirPath);
			} catch (e) {
				set({ error: (e as Error).message, isLoading: false });
			}
		},

		deleteFile: async (path: string) => {
			set({ isLoading: true, error: null });
			try {
				const response = await apiFetch(`/api/fs${path}`, {
					method: "DELETE",
				});
				if (!response.ok) {
					throw new Error(`Failed to delete file: ${response.statusText}`);
				}
				set({ selectedFile: null, fileContent: null, isLoading: false });
				// Reload current directory
				get().loadDirectory(get().currentPath);
			} catch (e) {
				set({ error: (e as Error).message, isLoading: false });
			}
		},

		loadGitStatus: async () => {
			set({ isLoading: true, error: null });
			try {
				const response = await apiFetch("/api/git/status");
				if (!response.ok) {
					throw new Error(`Failed to load git status: ${response.statusText}`);
				}
				const data = await response.json();
				set({ gitStatus: data, isLoading: false });
			} catch (e) {
				set({ error: (e as Error).message, isLoading: false });
			}
		},

		loadGitDiff: async () => {
			set({ isLoading: true, error: null });
			try {
				const response = await apiFetch("/api/git/diff");
				if (!response.ok) {
					throw new Error(`Failed to load git diff: ${response.statusText}`);
				}
				const data = await response.json();
				set({ gitDiff: data, isLoading: false });
			} catch (e) {
				set({ error: (e as Error).message, isLoading: false });
			}
		},

		setShowFileBrowser: (show: boolean) => {
			set({ showFileBrowser: show });
			if (show) {
				get().loadDirectory("/");
			}
		},

		setShowGitDiff: (show: boolean) => {
			set({ showGitDiff: show });
			if (show) {
				get().loadGitDiff();
			}
		},

		clearSelection: () => {
			set({ selectedFile: null, fileContent: null });
		},

		clearError: () => set({ error: null }),
	};
});
