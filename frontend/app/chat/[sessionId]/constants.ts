import type { AgentInfo } from './types';

// Agent 配置
export const agentInfoMap: Record<string, AgentInfo> = {
  assistant: {
    id: 'assistant',
    name: 'AI Assistant',
    description: '信息检索和事实核查',
    icon: '🔍',
    color: 'bg-blue-500',
    examples: [
      '什么是量子计算？',
      '请帮我查找关于人工智能的最新研究',
      '核查一下这个新闻是否准确...',
    ],
  },
};

// 文件上传限制（与后端保持一致）
export const MAX_IMAGE_COUNT = 4;
export const MAX_IMAGE_SIZE_BYTES = 5 * 1024 * 1024;
export const MAX_PDF_SIZE_BYTES = 20 * 1024 * 1024;
export const MAX_IMAGE_SIZE_MB = Math.round(MAX_IMAGE_SIZE_BYTES / (1024 * 1024));
export const MAX_PDF_SIZE_MB = Math.round(MAX_PDF_SIZE_BYTES / (1024 * 1024));

// 错误消息
export const IMAGE_COUNT_ERROR = `最多只能上传 ${MAX_IMAGE_COUNT} 个文件`;
export const IMAGE_SIZE_ERROR = `图片大小不能超过 ${MAX_IMAGE_SIZE_MB}MB`;
export const PDF_SIZE_ERROR = `PDF 大小不能超过 ${MAX_PDF_SIZE_MB}MB`;
export const IMAGE_TYPE_ERROR = '仅支持图片或 PDF 文件';
export const IMAGE_READ_ERROR = '读取文件失败';

// UI 常量
export const MAX_TEXTAREA_HEIGHT = 160; // 像素
export const SCROLL_RESET_DELAY = 100; // 毫秒
export const MESSAGES_PER_PAGE = 50;
export const LOAD_MORE_THRESHOLD = 100; // 像素
export const MIN_SCROLL_THRESHOLD = 5; // 像素
export const MIN_SCROLL_DISTANCE = 100; // 像素
export const SCROLL_DEBOUNCE_DELAY = 50; // 毫秒
export const FEEDBACK_TIMEOUT_MS = 2000; // 毫秒
