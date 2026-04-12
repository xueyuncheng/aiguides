package assistant

import (
	"aiguide/internal/pkg/constant"
	"fmt"
)

// toolCallLabel 将工具调用转换为用户可读的进度描述
func toolCallLabel(locale, name string, args map[string]any) string {
	if locale == constant.LocaleZH {
		return toolCallLabelZH(name, args)
	}

	return toolCallLabelEN(name, args)
}

func toolCallLabelZH(name string, args map[string]any) string {
	switch name {
	case "task_create":
		if title, ok := args["title"].(string); ok {
			return fmt.Sprintf("正在创建任务：%s", title)
		}
		return "正在创建任务"
	case "task_update":
		if id, ok := args["task_id"]; ok {
			return fmt.Sprintf("正在更新任务 #%v", id)
		}
		return "正在更新任务"
	case "task_list":
		return "正在检查任务列表"
	case "task_get":
		return "正在获取任务详情"
	case "finish_planning":
		return "规划完成，准备执行"
	case "web_search":
		if query, ok := args["query"].(string); ok {
			return fmt.Sprintf("正在搜索：%s", query)
		}
		return "正在搜索"
	case "exa_search":
		if query, ok := args["query"].(string); ok {
			return fmt.Sprintf("正在语义搜索：%s", query)
		}
		return "正在语义搜索"
	case "web_fetch":
		if url, ok := args["url"].(string); ok {
			return fmt.Sprintf("正在获取网页：%s", url)
		}
		return "正在获取网页"
	case "file_download":
		if url, ok := args["url"].(string); ok {
			return fmt.Sprintf("正在下载文件：%s", url)
		}
		return "正在下载文件"
	case "file_list":
		return "正在查看文件列表"
	case "file_get":
		return "正在获取文件信息"
	case "pdf_extract_text":
		return "正在提取 PDF 文本"
	case "pdf_generate_document":
		return "正在生成 PDF 文档"
	case "image_gen":
		return "正在生成图片"
	case "email_query":
		return "正在查询邮件"
	case "current_time":
		return "正在获取当前时间"
	case "audio_transcribe":
		return "正在转写音频"
	case "manage_memory":
		return "正在管理记忆"
	default:
		return fmt.Sprintf("调用 %s", name)
	}
}

func toolCallLabelEN(name string, args map[string]any) string {
	switch name {
	case "task_create":
		if title, ok := args["title"].(string); ok {
			return fmt.Sprintf("Creating task: %s", title)
		}
		return "Creating task"
	case "task_update":
		if id, ok := args["task_id"]; ok {
			return fmt.Sprintf("Updating task #%v", id)
		}
		return "Updating task"
	case "task_list":
		return "Checking task list"
	case "task_get":
		return "Getting task details"
	case "finish_planning":
		return "Planning complete, preparing to execute"
	case "web_search":
		if query, ok := args["query"].(string); ok {
			return fmt.Sprintf("Searching: %s", query)
		}
		return "Searching"
	case "exa_search":
		if query, ok := args["query"].(string); ok {
			return fmt.Sprintf("Semantic search: %s", query)
		}
		return "Semantic search"
	case "web_fetch":
		if url, ok := args["url"].(string); ok {
			return fmt.Sprintf("Fetching web page: %s", url)
		}
		return "Fetching web page"
	case "file_download":
		if url, ok := args["url"].(string); ok {
			return fmt.Sprintf("Downloading file: %s", url)
		}
		return "Downloading file"
	case "file_list":
		return "Listing files"
	case "file_get":
		return "Getting file details"
	case "pdf_extract_text":
		return "Extracting PDF text"
	case "pdf_generate_document":
		return "Generating PDF document"
	case "image_gen":
		return "Generating image"
	case "email_query":
		return "Querying email"
	case "current_time":
		return "Getting current time"
	case "audio_transcribe":
		return "Transcribing audio"
	case "manage_memory":
		return "Managing memory"
	default:
		return fmt.Sprintf("Calling %s", name)
	}
}
