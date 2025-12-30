package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// MailFetchInput 定义邮件获取的输入参数
type MailFetchInput struct {
	MaxCount int    `json:"max_count" jsonschema:"要获取的最大邮件数量，默认为 10"`
	Mailbox  string `json:"mailbox,omitempty" jsonschema:"邮箱名称（可选），例如 'INBOX' 或 '收件箱'，默认获取收件箱"`
}

// EmailMessage 定义单封邮件的结构
type EmailMessage struct {
	Subject string `json:"subject"`
	From    string `json:"from"`
	Date    string `json:"date"`
	Content string `json:"content"`
	IsRead  bool   `json:"is_read"`
	Mailbox string `json:"mailbox"`
}

// MailFetchOutput 定义邮件获取的输出
type MailFetchOutput struct {
	Success bool           `json:"success"`
	Emails  []EmailMessage `json:"emails,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// NewMailFetchToolWithJSON 创建 Apple Mail 邮件获取工具。
//
// 该工具通过 AppleScript 从 Apple Mail 客户端读取邮件，
// 使用 JSON 格式返回邮件列表用于总结分析。
func NewMailFetchToolWithJSON() (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "fetch_apple_mail",
		Description: "从 Apple Mail 客户端获取邮件。返回邮件列表，包括主题、发件人、日期、内容等信息，适用于邮件分析和总结任务。",
	}

	handler := func(ctx tool.Context, input MailFetchInput) (*MailFetchOutput, error) {
		return fetchAppleMailWithJSON(ctx, input), nil
	}

	return functiontool.New(config, handler)
}

// fetchAppleMailWithJSON 使用 JSON 输出格式的 AppleScript
func fetchAppleMailWithJSON(ctx context.Context, input MailFetchInput) *MailFetchOutput {
	// 检查是否在 macOS 上运行
	if runtime.GOOS != "darwin" {
		return &MailFetchOutput{
			Success: false,
			Error:   "此工具仅支持 macOS 系统的 Apple Mail",
		}
	}

	// 设置默认值
	maxCount := input.MaxCount
	if maxCount <= 0 {
		maxCount = 10
	}
	if maxCount > 50 {
		maxCount = 50
	}

	mailbox := input.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	// 构建 AppleScript 脚本，直接生成 JSON 格式输出
	// 使用 AppleScript 的文本替换功能进行基本的 JSON 转义
	simpleScript := fmt.Sprintf(`
tell application "Mail"
	set output to "["
	set targetMailbox to mailbox "%s"
	set messageList to messages of targetMailbox
	
	set messageCount to count of messageList
	if messageCount > %d then
		set messageCount to %d
	end if
	
	repeat with i from 1 to messageCount
		set theMessage to item i of messageList
		set emailSubject to subject of theMessage
		set emailSender to sender of theMessage
		set emailDate to date received of theMessage as string
		set emailContent to content of theMessage
		set emailRead to read status of theMessage
		
		-- 限制内容长度
		if length of emailContent > 500 then
			set emailContent to text 1 thru 500 of emailContent
		end if
		
		-- 转义 JSON 特殊字符
		-- 注意：AppleScript 中的转义有限，这里处理最常见的情况
		-- 对于复杂内容，可能需要在 Go 侧进行额外的清理
		set emailSubject to my replaceText(emailSubject, "\\", "\\\\")
		set emailSubject to my replaceText(emailSubject, quote, "\\\"")
		set emailSubject to my replaceText(emailSubject, tab, " ")
		set emailSubject to my replaceText(emailSubject, return, " ")
		set emailSubject to my replaceText(emailSubject, linefeed, " ")
		
		set emailContent to my replaceText(emailContent, "\\", "\\\\")
		set emailContent to my replaceText(emailContent, quote, "\\\"")
		set emailContent to my replaceText(emailContent, tab, " ")
		set emailContent to my replaceText(emailContent, return, " ")
		set emailContent to my replaceText(emailContent, linefeed, " ")
		
		if emailRead then
			set readStatus to "true"
		else
			set readStatus to "false"
		end if
		
		set emailJSON to "{\"subject\":\"" & emailSubject & "\",\"from\":\"" & emailSender & "\",\"date\":\"" & emailDate & "\",\"content\":\"" & emailContent & "\",\"is_read\":" & readStatus & ",\"mailbox\":\"%s\"}"
		
		set output to output & emailJSON
		if i < messageCount then
			set output to output & ","
		end if
	end repeat
	
	set output to output & "]"
	return output
end tell

on replaceText(theText, oldText, newText)
	set AppleScript's text item delimiters to oldText
	set theItems to text items of theText
	set AppleScript's text item delimiters to newText
	set theText to theItems as string
	set AppleScript's text item delimiters to ""
	return theText
end replaceText
`, mailbox, maxCount, maxCount, mailbox)

	// 执行 AppleScript
	cmd := exec.CommandContext(ctx, "osascript", "-e", simpleScript)
	output, err := cmd.Output()
	if err != nil {
		// 如果执行失败，可能是因为权限或 Mail 应用未运行
		return &MailFetchOutput{
			Success: false,
			Error:   fmt.Sprintf("执行 AppleScript 失败: %v，请确保 Mail 应用正在运行且已授予权限", err),
		}
	}

	// 解析 JSON 输出
	var emails []EmailMessage
	if err := json.Unmarshal(output, &emails); err != nil {
		return &MailFetchOutput{
			Success: false,
			Error:   fmt.Sprintf("解析邮件 JSON 失败: %v，输出: %s", err, string(output)),
		}
	}

	return &MailFetchOutput{
		Success: true,
		Emails:  emails,
	}
}
