// Message types
export interface Message {
	id: string;
	role: "user" | "assistant" | "system";
	content: string;
	toolCalls?: ToolCall[];
	timestamp: Date;
}

export interface ToolCall {
	id: string;
	name: string;
	input?: Record<string, unknown>;
	output?: string;
	status: "pending" | "completed" | "error";
}

// History message from server
export interface HistoryMessage {
	id: string;
	role: "user" | "assistant" | "system";
	content: string;
	tool_calls?: HistoryToolCall[];
	timestamp: string;
}

export interface HistoryToolCall {
	id: string;
	name: string;
	input?: Record<string, unknown>;
	output?: string;
	status: string;
}

// Session types
export interface Session {
	id: string;
	title: string;
	createdAt: string;
	updatedAt: string;
}

// Permission request
export interface PermissionRequest {
	permissionId: string;
	toolName: string;
	description: string;
}

// User question
export interface UserQuestion {
	questionId: string;
	question: string;
	options: QuestionOption[];
}

export interface QuestionOption {
	label: string;
	description?: string;
}

// Connection state
export type ConnectionState =
	| "disconnected"
	| "connecting"
	| "connected"
	| "authenticated";
