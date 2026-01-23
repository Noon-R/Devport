import { create } from "zustand";
import type {
	ConnectionState,
	HistoryMessage,
	Message,
	PermissionRequest,
	Session,
	ToolCall,
	UserQuestion,
} from "./types";

interface WsState {
	// Connection state
	connectionState: ConnectionState;
	error: string | null;

	// Connection info (for reconnect)
	wsUrl: string | null;
	authToken: string | null;

	// Session
	currentSessionId: string | null;
	sessions: Session[];

	// Messages
	messages: Message[];
	isGenerating: boolean;
	lastMessageId: string | null;

	// Pending interactions
	pendingPermission: PermissionRequest | null;
	pendingQuestion: UserQuestion | null;

	// Actions
	connect: (url: string, token: string) => Promise<void>;
	disconnect: () => void;
	sendMessage: (content: string) => Promise<void>;
	interrupt: () => Promise<void>;
	attachSession: (sessionId: string) => Promise<void>;
	createSession: (title?: string) => Promise<Session>;
	loadSessions: () => Promise<void>;
	respondToPermission: (allowed: boolean) => Promise<void>;
	respondToQuestion: (answer: string) => Promise<void>;
	clearError: () => void;
	syncMessages: () => Promise<void>;
}

let ws: WebSocket | null = null;
let reconnectAttempts = 0;
let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
const MAX_RECONNECT_ATTEMPTS = 10;
const BASE_RECONNECT_DELAY = 1000;

export const useWsStore = create<WsState>((set, get) => {
	let currentAssistantMessage: Message | null = null;
	let rpcRequestId = 0;
	const pendingRequests = new Map<
		number,
		{ resolve: (value: unknown) => void; reject: (error: Error) => void }
	>();

	// Get HTTP base URL from WebSocket URL
	const getHttpUrl = (): string => {
		const { wsUrl } = get();
		if (!wsUrl) return "";
		return wsUrl
			.replace("ws://", "http://")
			.replace("wss://", "https://")
			.replace("/ws", "");
	};

	// Fetch wrapper with auth
	const apiFetch = async (
		endpoint: string,
		options?: RequestInit,
	): Promise<Response> => {
		const { authToken } = get();
		const url = `${getHttpUrl()}${endpoint}`;
		return fetch(url, {
			...options,
			headers: {
				Authorization: `Bearer ${authToken}`,
				"Content-Type": "application/json",
				...options?.headers,
			},
		});
	};

	// Handle notifications from server
	const handleNotification = (
		method: string,
		params: Record<string, unknown>,
	) => {
		const sessionId = params.session_id as string;
		if (sessionId !== get().currentSessionId) return;

		switch (method) {
			case "chat.text": {
				const content = params.content as string;
				if (!currentAssistantMessage) {
					currentAssistantMessage = {
						id: crypto.randomUUID(),
						role: "assistant",
						content: "",
						toolCalls: [],
						timestamp: new Date(),
					};
					set((state) => ({
						messages: [...state.messages, currentAssistantMessage!],
					}));
				}
				currentAssistantMessage.content += content;
				set((state) => ({
					messages: state.messages.map((m) =>
						m.id === currentAssistantMessage!.id
							? { ...m, content: currentAssistantMessage!.content }
							: m,
					),
				}));
				break;
			}

			case "chat.tool_call": {
				if (currentAssistantMessage) {
					const toolCall: ToolCall = {
						id: params.tool_use_id as string,
						name: params.tool_name as string,
						input: params.input as Record<string, unknown>,
						status: "pending",
					};
					currentAssistantMessage.toolCalls = [
						...(currentAssistantMessage.toolCalls || []),
						toolCall,
					];
					set((state) => ({
						messages: state.messages.map((m) =>
							m.id === currentAssistantMessage!.id
								? { ...m, toolCalls: currentAssistantMessage!.toolCalls }
								: m,
						),
					}));
				}
				break;
			}

			case "chat.tool_result": {
				if (currentAssistantMessage) {
					const toolId = params.tool_use_id as string;
					const output = params.output as string;
					currentAssistantMessage.toolCalls =
						currentAssistantMessage.toolCalls?.map((tc) =>
							tc.id === toolId
								? { ...tc, output, status: "completed" as const }
								: tc,
						);
					set((state) => ({
						messages: state.messages.map((m) =>
							m.id === currentAssistantMessage!.id
								? { ...m, toolCalls: currentAssistantMessage!.toolCalls }
								: m,
						),
					}));
				}
				break;
			}

			case "chat.permission_request": {
				set({
					pendingPermission: {
						permissionId: params.permission_id as string,
						toolName: params.tool_name as string,
						description: params.description as string,
					},
				});
				break;
			}

			case "chat.ask_user_question": {
				set({
					pendingQuestion: {
						questionId: params.question_id as string,
						question: params.question as string,
						options: params.options as {
							label: string;
							description?: string;
						}[],
					},
				});
				break;
			}

			case "chat.done": {
				const msgId = currentAssistantMessage?.id || null;
				currentAssistantMessage = null;
				set({ isGenerating: false, lastMessageId: msgId });
				break;
			}

			case "chat.error": {
				set({
					isGenerating: false,
					error: params.error as string,
				});
				break;
			}

			case "chat.interrupted": {
				currentAssistantMessage = null;
				set({ isGenerating: false });
				break;
			}

			case "chat.system": {
				const systemMessage: Message = {
					id: crypto.randomUUID(),
					role: "system",
					content: params.message as string,
					timestamp: new Date(),
				};
				set((state) => ({
					messages: [...state.messages, systemMessage],
				}));
				break;
			}
		}
	};

	const sendRpcRequest = (
		method: string,
		params: Record<string, unknown>,
	): Promise<unknown> => {
		return new Promise((resolve, reject) => {
			if (!ws || ws.readyState !== WebSocket.OPEN) {
				reject(new Error("WebSocket not connected"));
				return;
			}

			const id = ++rpcRequestId;
			pendingRequests.set(id, { resolve, reject });

			ws.send(
				JSON.stringify({
					jsonrpc: "2.0",
					method,
					params,
					id,
				}),
			);
		});
	};

	// Attempt reconnection
	const attemptReconnect = () => {
		const { wsUrl, authToken, currentSessionId } = get();
		if (!wsUrl || !authToken) return;

		if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
			set({ error: "Failed to reconnect after multiple attempts" });
			return;
		}

		const delay = Math.min(
			BASE_RECONNECT_DELAY * 2 ** reconnectAttempts,
			30000,
		);
		reconnectAttempts++;

		reconnectTimeout = setTimeout(async () => {
			try {
				await get().connect(wsUrl, authToken);
				// Reattach to session if we had one
				if (currentSessionId) {
					await get().attachSession(currentSessionId);
					// Sync any missing messages
					await get().syncMessages();
				}
				reconnectAttempts = 0;
			} catch {
				attemptReconnect();
			}
		}, delay);
	};

	return {
		connectionState: "disconnected",
		error: null,
		wsUrl: null,
		authToken: null,
		currentSessionId: null,
		sessions: [],
		messages: [],
		isGenerating: false,
		lastMessageId: null,
		pendingPermission: null,
		pendingQuestion: null,

		connect: async (url: string, token: string) => {
			// Clear any pending reconnect
			if (reconnectTimeout) {
				clearTimeout(reconnectTimeout);
				reconnectTimeout = null;
			}

			set({
				connectionState: "connecting",
				error: null,
				wsUrl: url,
				authToken: token,
			});

			return new Promise((resolve, reject) => {
				ws = new WebSocket(url);

				ws.onopen = async () => {
					set({ connectionState: "connected" });

					try {
						await sendRpcRequest("auth", { token });
						set({ connectionState: "authenticated" });
						reconnectAttempts = 0;
						resolve();
					} catch (e) {
						set({
							connectionState: "disconnected",
							error: "Authentication failed",
						});
						reject(e);
					}
				};

				ws.onmessage = (event) => {
					const data = JSON.parse(event.data);

					// Notification (no id)
					if (!("id" in data) && data.method) {
						handleNotification(data.method, data.params || {});
						return;
					}

					// Response
					if (data.id && pendingRequests.has(data.id)) {
						const { resolve, reject } = pendingRequests.get(data.id)!;
						pendingRequests.delete(data.id);

						if (data.error) {
							reject(new Error(data.error.message));
						} else {
							resolve(data.result);
						}
					}
				};

				ws.onclose = () => {
					const wasAuthenticated = get().connectionState === "authenticated";
					set({ connectionState: "disconnected" });
					pendingRequests.clear();

					// Attempt reconnect if we were connected
					if (wasAuthenticated) {
						attemptReconnect();
					}
				};

				ws.onerror = () => {
					set({
						connectionState: "disconnected",
						error: "Connection failed",
					});
					reject(new Error("Connection failed"));
				};
			});
		},

		disconnect: () => {
			// Clear any pending reconnect
			if (reconnectTimeout) {
				clearTimeout(reconnectTimeout);
				reconnectTimeout = null;
			}
			reconnectAttempts = MAX_RECONNECT_ATTEMPTS; // Prevent auto-reconnect

			ws?.close();
			ws = null;
			set({
				connectionState: "disconnected",
				currentSessionId: null,
				messages: [],
				wsUrl: null,
				authToken: null,
			});
		},

		sendMessage: async (content: string) => {
			const { currentSessionId } = get();
			if (!currentSessionId) return;

			// Add user message
			const userMessage: Message = {
				id: crypto.randomUUID(),
				role: "user",
				content,
				timestamp: new Date(),
			};
			set((state) => ({
				messages: [...state.messages, userMessage],
				isGenerating: true,
			}));

			try {
				// Try WebSocket first
				if (ws && ws.readyState === WebSocket.OPEN) {
					await sendRpcRequest("chat.message", {
						session_id: currentSessionId,
						content,
					});
				} else {
					// Fallback to REST API
					const response = await apiFetch(
						`/api/sessions/${currentSessionId}/messages`,
						{
							method: "POST",
							body: JSON.stringify({ content }),
						},
					);
					if (!response.ok) {
						throw new Error("Failed to send message");
					}
				}
			} catch (e) {
				set({
					isGenerating: false,
					error: (e as Error).message,
				});
			}
		},

		interrupt: async () => {
			const { currentSessionId } = get();
			if (!currentSessionId) return;

			try {
				// Try WebSocket first
				if (ws && ws.readyState === WebSocket.OPEN) {
					await sendRpcRequest("chat.interrupt", {
						session_id: currentSessionId,
					});
				} else {
					// Fallback to REST API
					await apiFetch(`/api/sessions/${currentSessionId}/cancel`, {
						method: "POST",
					});
				}
			} catch (e) {
				set({ error: (e as Error).message });
			}
		},

		attachSession: async (sessionId: string) => {
			try {
				const result = (await sendRpcRequest("chat.attach", {
					session_id: sessionId,
				})) as {
					session_id: string;
					status: string;
					history: HistoryMessage[];
				};

				// Convert history messages to local Message format
				const messages: Message[] = (result.history || []).map((hm) => ({
					id: hm.id,
					role: hm.role,
					content: hm.content,
					toolCalls: hm.tool_calls?.map((tc) => ({
						id: tc.id,
						name: tc.name,
						input: tc.input,
						output: tc.output,
						status: tc.status as "pending" | "completed" | "error",
					})),
					timestamp: new Date(hm.timestamp),
				}));

				const lastId =
					messages.length > 0 ? messages[messages.length - 1].id : null;

				set({
					currentSessionId: sessionId,
					messages,
					lastMessageId: lastId,
					pendingPermission: null,
					pendingQuestion: null,
				});
			} catch (e) {
				set({ error: (e as Error).message });
			}
		},

		createSession: async (title?: string) => {
			const result = (await sendRpcRequest("session.create", {
				title: title || "New Chat",
			})) as { session: Session };
			const session = result.session;
			set((state) => ({
				sessions: [...state.sessions, session],
			}));
			return session;
		},

		loadSessions: async () => {
			try {
				const result = (await sendRpcRequest("session.list", {})) as {
					sessions: Session[];
				};
				set({ sessions: result.sessions || [] });
			} catch (e) {
				set({ error: (e as Error).message });
			}
		},

		respondToPermission: async (allowed: boolean) => {
			const { currentSessionId, pendingPermission } = get();
			if (!currentSessionId || !pendingPermission) return;

			try {
				// Try WebSocket first
				if (ws && ws.readyState === WebSocket.OPEN) {
					await sendRpcRequest("chat.permission_response", {
						session_id: currentSessionId,
						permission_id: pendingPermission.permissionId,
						allowed,
					});
				} else {
					// Fallback to REST API
					await apiFetch(`/api/permissions/${pendingPermission.permissionId}`, {
						method: "POST",
						body: JSON.stringify({
							session_id: currentSessionId,
							allowed,
						}),
					});
				}
				set({ pendingPermission: null });
			} catch (e) {
				set({ error: (e as Error).message });
			}
		},

		respondToQuestion: async (answer: string) => {
			const { currentSessionId, pendingQuestion } = get();
			if (!currentSessionId || !pendingQuestion) return;

			try {
				// Try WebSocket first
				if (ws && ws.readyState === WebSocket.OPEN) {
					await sendRpcRequest("chat.question_response", {
						session_id: currentSessionId,
						question_id: pendingQuestion.questionId,
						answer,
					});
				} else {
					// Fallback to REST API
					await apiFetch(`/api/questions/${pendingQuestion.questionId}`, {
						method: "POST",
						body: JSON.stringify({
							session_id: currentSessionId,
							answer,
						}),
					});
				}
				set({ pendingQuestion: null });
			} catch (e) {
				set({ error: (e as Error).message });
			}
		},

		// Sync messages after reconnection
		syncMessages: async () => {
			const { currentSessionId, lastMessageId } = get();
			if (!currentSessionId) return;

			try {
				let url = `/api/sessions/${currentSessionId}/messages`;
				if (lastMessageId) {
					url += `?after=${lastMessageId}`;
				}

				const response = await apiFetch(url);
				if (!response.ok) return;

				const data = await response.json();
				const newMessages: Message[] = (data.messages || []).map(
					(hm: HistoryMessage) => ({
						id: hm.id,
						role: hm.role,
						content: hm.content,
						toolCalls: hm.tool_calls?.map((tc) => ({
							id: tc.id,
							name: tc.name,
							input: tc.input,
							output: tc.output,
							status: tc.status as "pending" | "completed" | "error",
						})),
						timestamp: new Date(hm.timestamp),
					}),
				);

				if (newMessages.length > 0) {
					set((state) => ({
						messages: [...state.messages, ...newMessages],
						lastMessageId: newMessages[newMessages.length - 1].id,
					}));
				}
			} catch {
				// Silently fail sync
			}
		},

		clearError: () => set({ error: null }),
	};
});
