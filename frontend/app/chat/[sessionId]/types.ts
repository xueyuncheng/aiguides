export interface ToolCallItem {
  toolCallId?: string;
  toolName: string;
  label: string;
  args?: Record<string, unknown>;
  result?: Record<string, unknown>;
  status?: 'running' | 'completed';
}

export interface ToolCallResponse {
  tool_call_id?: string;
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
  videos?: string[];
  fileNames?: string[];
  files?: MessageFile[];
  isError?: boolean;
  toolCalls?: ToolCallItem[];
  voiceAudioFileId?: number;
  voiceAudioUrl?: string;
  isVoiceMessage?: boolean;
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
  mimeType?: string;
  isPdf?: boolean;
  isAudio?: boolean;
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
  videos?: string[];
  file_names?: string[];
  files?: MessageFile[];
  tool_calls?: ToolCallResponse[];
  voice_audio_file_id?: number;
}

export interface SessionHistoryResponse {
  messages: HistoryMessageResponse[];
  total?: number;
  has_more?: boolean;
}
