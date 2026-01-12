package tools

import (
	"fmt"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/agenttool"
)

// NewEmailAgent 创建邮件查询 Agent
// 该 Agent 封装了邮件查询工具，使其能够更好地理解用户的自然语言查询需求。
// 通过 Agent 封装，可以让 LLM 对邮件查询请求进行推理，例如理解"重要邮件"、"最近的邮件"等自然语言表达。
func NewEmailAgent(model model.LLM) (tool.Tool, error) {
	// 创建邮件查询工具
	emailQueryTool, err := NewEmailQueryTool()
	if err != nil {
		return nil, fmt.Errorf("NewEmailQueryTool() error: %w", err)
	}

	// 创建一个专门负责邮件查询的子 Agent
	emailAgent, err := llmagent.New(llmagent.Config{
		Name:        "email_query",
		Description: "查询和搜索邮箱中的邮件。支持通过 IMAP 协议连接邮件服务器，查询指定邮箱文件夹中的邮件列表。可以查询所有邮件或仅查询未读邮件，并可限制返回的邮件数量。",
		Model:       model,
		Instruction: `你是一个专业的邮件查询助手。你的职责是帮助用户查询和管理他们的邮件。

当用户请求查询邮件时，你需要：
1. 理解用户的查询意图（例如："最近的邮件"、"未读邮件"、"重要邮件"等）
2. 使用 query_emails 工具查询邮件
3. 将查询结果以清晰、有条理的方式呈现给用户

注意事项：
- 默认查询 INBOX 文件夹中的最新 10 封邮件
- 如果用户只想看未读邮件，设置 unseen=true
- 如果用户想看更多邮件，可以增加 limit 参数（最多 50）
- 如果用户提到特定的邮箱文件夹（如 Sent、Drafts），使用相应的 mailbox 参数
- 对于查询结果，重点关注邮件的主题、发件人、日期和是否已读
- 如果邮件内容很长，可以适当总结

请直接使用邮件查询工具来完成用户的请求，不需要过多的解释，专注于提供准确、有用的邮件信息。`,
		Tools: []tool.Tool{
			emailQueryTool,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("llmagent.New() for email agent error: %w", err)
	}

	// 使用 agenttool 将该 Agent 封装成一个工具
	// 这样主 Agent 看到的将是一个名为 email_query 的普通工具
	emailAgentTool := agenttool.New(emailAgent, nil)

	return emailAgentTool, nil
}
