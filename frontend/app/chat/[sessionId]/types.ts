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
