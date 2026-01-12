package tools

import (
	_ "embed"
	"fmt"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/agenttool"
)

//go:embed email_agent_prompt.md
var emailAgentInstruction string

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
		Instruction: emailAgentInstruction,
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
