package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

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

// NewMailFetchTool 使用 functiontool.New 创建 Apple Mail 邮件获取工具。
//
// 该工具通过 AppleScript 从 Apple Mail 客户端读取邮件，返回邮件列表用于总结分析。
func NewMailFetchTool() (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "fetch_apple_mail",
		Description: "从 Apple Mail 客户端获取邮件。返回邮件列表，包括主题、发件人、日期、内容等信息，适用于邮件分析和总结任务。",
	}

	handler := func(ctx tool.Context, input MailFetchInput) (*MailFetchOutput, error) {
		return fetchAppleMail(ctx, input), nil
	}

	return functiontool.New(config, handler)
}

// fetchAppleMail 执行实际的邮件获取逻辑
func fetchAppleMail(ctx context.Context, input MailFetchInput) *MailFetchOutput {
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
		maxCount = 50 // 限制最大数量
	}

	mailbox := input.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	// 构建 AppleScript 脚本
	script := fmt.Sprintf(`
tell application "Mail"
	set emailList to {}
	set targetMailbox to mailbox "%s"
	set messageList to messages of targetMailbox
	
	-- 限制获取的邮件数量
	set messageCount to count of messageList
	if messageCount > %d then
		set messageCount to %d
	end if
	
	repeat with i from 1 to messageCount
		set theMessage to item i of messageList
		set emailInfo to {subject:subject of theMessage, sender:sender of theMessage, dateReceived:date received of theMessage, content:content of theMessage, isRead:read status of theMessage, mailboxName:"%s"}
		set end of emailList to emailInfo
	end repeat
	
	return emailList
end tell
`, mailbox, maxCount, maxCount, mailbox)

	// 执行 AppleScript
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return &MailFetchOutput{
			Success: false,
			Error:   fmt.Sprintf("执行 AppleScript 失败: %v", err),
		}
	}

	// 解析 AppleScript 输出
	emails, parseErr := parseAppleScriptOutput(string(output))
	if parseErr != nil {
		return &MailFetchOutput{
			Success: false,
			Error:   fmt.Sprintf("解析邮件数据失败: %v", parseErr),
		}
	}

	return &MailFetchOutput{
		Success: true,
		Emails:  emails,
	}
}

// parseAppleScriptOutput 解析 AppleScript 返回的邮件列表
// AppleScript 返回的格式类似于：{subject:"Test", sender:"user@example.com", ...}
func parseAppleScriptOutput(output string) ([]EmailMessage, error) {
	// 简化的解析逻辑
	// AppleScript 的输出格式比较复杂，这里提供基本的解析
	// 实际使用中可能需要更复杂的解析逻辑

	output = strings.TrimSpace(output)
	if output == "" || output == "{}" {
		return []EmailMessage{}, nil
	}

	// 尝试使用更简单的方法：使用 JSON 格式化的 AppleScript
	// 这里先返回一个基本的解析结果
	// 在实际应用中，可能需要使用更复杂的 AppleScript 输出 JSON 格式

	var emails []EmailMessage

	// 简单解析（这是一个占位实现，实际需要根据 AppleScript 的具体输出格式调整）
	// 由于 AppleScript 的输出格式复杂，建议修改 AppleScript 直接输出 JSON

	// 暂时返回解析错误，提示需要改进
	return emails, fmt.Errorf("AppleScript 输出解析需要根据实际输出格式实现，输出内容: %s", output)
}

// NewMailFetchToolWithJSON 使用 JSON 输出的 AppleScript 版本
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

	// 使用 Python 辅助生成 JSON（macOS 自带 Python）
	// 注释掉未使用的 Python 方案，保留供将来参考
	/*
		script := fmt.Sprintf(`
	import json
	import subprocess

	applescript = '''
	tell application "Mail"
		set emailList to {}
		try
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
				if length of emailContent > 1000 then
					set emailContent to text 1 thru 1000 of emailContent
				end if

				set emailInfo to {subject:emailSubject, sender:emailSender, dateReceived:emailDate, content:emailContent, isRead:emailRead}
				set end of emailList to emailInfo
			end repeat
		end try

		return emailList
	end tell
	'''

	result = subprocess.run(['osascript', '-e', applescript], capture_output=True, text=True)
	# AppleScript 输出需要解析为 JSON
	# 这里仅返回原始输出
	print(result.stdout)
	`, mailbox, maxCount, maxCount)
	*/

	// 由于解析复杂，我们采用更简单的方案：直接在 AppleScript 中限制输出
	// 并提供一个简化版本供测试和开发
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
		
		-- 转义 JSON 特殊字符（简化版）
		set emailSubject to my replaceText(emailSubject, quote, "\\\"")
		set emailContent to my replaceText(emailContent, quote, "\\\"")
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
