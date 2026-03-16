export interface ToolCallItem {
  toolName: string;
  label: string;
  args?: Record<string, unknown>;
  result?: Record<string, unknown>;
}

export interface ToolCallResponse {
  tool_name: string;
  label: string;
  args?: Record<string, unknown>;
  result?: Record<string, unknown>;
}

export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  thought?: string;
  timestamp: Date;
  author?: string;
  isStreaming?: boolean;
  images?: string[];
  fileNames?: string[]; // 文件名列表，与 images 对应
  isError?: boolean;
  toolCalls?: ToolCallItem[];
}

export interface SelectedImage {
  id: string;
  dataUrl: string;
  name: string;
  isPdf?: boolean;
}

export interface AgentInfo {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
  examples: string[];
}

export interface ChatRequest {
  user_id: number;
  session_id: string;
  message: string;
  images: string[];
  file_names: string[];
}

export interface HistoryMessageResponse {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  thought?: string;
  timestamp: string;
  images?: string[];
  file_names?: string[];
  tool_calls?: ToolCallResponse[];
}

export interface SessionHistoryResponse {
  messages: HistoryMessageResponse[];
  total?: number;
  has_more?: boolean;
}
