import { create } from "zustand";
import type {
	ConnectionState,
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

	// Session
	currentSessionId: string | null;
	sessions: Session[];

	// Messages
	messages: Message[];
	isGenerating: boolean;

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
}

let ws: WebSocket | null = null;

export const useWsStore = create<WsState>((set, get) => {
	let currentAssistantMessage: Message | null = null;
	let rpcRequestId = 0;
	const pendingRequests = new Map<
		number,
		{ resolve: (value: unknown) => void; reject: (error: Error) => void }
	>();

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
				currentAssistantMessage = null;
				set({ isGenerating: false });
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

	return {
		connectionState: "disconnected",
		error: null,
		currentSessionId: null,
		sessions: [],
		messages: [],
		isGenerating: false,
		pendingPermission: null,
		pendingQuestion: null,

		connect: async (url: string, token: string) => {
			set({ connectionState: "connecting", error: null });

			return new Promise((resolve, reject) => {
				ws = new WebSocket(url);

				ws.onopen = async () => {
					set({ connectionState: "connected" });

					try {
						await sendRpcRequest("auth", { token });
						set({ connectionState: "authenticated" });
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
					set({ connectionState: "disconnected" });
					pendingRequests.clear();
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
			ws?.close();
			ws = null;
			set({
				connectionState: "disconnected",
				currentSessionId: null,
				messages: [],
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
				await sendRpcRequest("chat.message", {
					session_id: currentSessionId,
					content,
				});
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
				await sendRpcRequest("chat.interrupt", {
					session_id: currentSessionId,
				});
			} catch (e) {
				set({ error: (e as Error).message });
			}
		},

		attachSession: async (sessionId: string) => {
			try {
				await sendRpcRequest("chat.attach", { session_id: sessionId });
				set({
					currentSessionId: sessionId,
					messages: [],
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
				await sendRpcRequest("chat.permission_response", {
					session_id: currentSessionId,
					permission_id: pendingPermission.permissionId,
					allowed,
				});
				set({ pendingPermission: null });
			} catch (e) {
				set({ error: (e as Error).message });
			}
		},

		respondToQuestion: async (answer: string) => {
			const { currentSessionId, pendingQuestion } = get();
			if (!currentSessionId || !pendingQuestion) return;

			try {
				await sendRpcRequest("chat.question_response", {
					session_id: currentSessionId,
					question_id: pendingQuestion.questionId,
					answer,
				});
				set({ pendingQuestion: null });
			} catch (e) {
				set({ error: (e as Error).message });
			}
		},

		clearError: () => set({ error: null }),
	};
});
