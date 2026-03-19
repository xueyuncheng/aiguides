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
  files?: MessageFile[];
  isError?: boolean;
  toolCalls?: ToolCallItem[];
}

export interface MessageFile {
  mime_type: string;
  name?: string;
  label?: string;
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
  files?: MessageFile[];
  tool_calls?: ToolCallResponse[];
}

export interface SessionHistoryResponse {
  messages: HistoryMessageResponse[];
  total?: number;
  has_more?: boolean;
}
