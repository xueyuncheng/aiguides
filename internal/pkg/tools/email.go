package tools

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// EmailQueryInput 定义邮件查询工具的输入参数
type EmailQueryInput struct {
	Server   string `json:"server" jsonschema:"IMAP 服务器地址，例如: imap.gmail.com:993"`
	Username string `json:"username" jsonschema:"邮箱账号"`
	Password string `json:"password" jsonschema:"邮箱密码或应用专用密码"`
	Mailbox  string `json:"mailbox,omitempty" jsonschema:"邮箱文件夹名称，默认为 INBOX"`
	Limit    int    `json:"limit,omitempty" jsonschema:"返回的邮件数量限制，默认为 10，最多 50"`
	Unseen   bool   `json:"unseen,omitempty" jsonschema:"是否只查询未读邮件，默认为 false"`
}

// EmailMessage 表示一封邮件的基本信息
type EmailMessage struct {
	UID      uint32    `json:"uid"`
	Subject  string    `json:"subject"`
	From     string    `json:"from"`
	To       string    `json:"to"`
	Date     time.Time `json:"date"`
	Seen     bool      `json:"seen"`
	BodyText string    `json:"body_text,omitempty"`
}

// EmailQueryOutput 定义邮件查询工具的输出
type EmailQueryOutput struct {
	Success  bool           `json:"success"`
	Messages []EmailMessage `json:"messages,omitempty"`
	Count    int            `json:"count"`
	Message  string         `json:"message,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// NewEmailQueryTool 创建邮件查询工具
//
// 该工具使用 IMAP 协议连接邮件服务器并查询邮件
func NewEmailQueryTool() (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "query_emails",
		Description: "查询邮箱中的邮件。支持通过 IMAP 协议连接邮件服务器，查询指定邮箱文件夹中的邮件列表。可以查询所有邮件或仅查询未读邮件，并可限制返回的邮件数量。",
	}

	handler := func(ctx tool.Context, input EmailQueryInput) (*EmailQueryOutput, error) {
		return queryEmails(ctx, input)
	}

	return functiontool.New(config, handler)
}

// queryEmails 查询邮件
func queryEmails(ctx context.Context, input EmailQueryInput) (*EmailQueryOutput, error) {
	// 验证必填参数
	if input.Server == "" {
		slog.Error("IMAP 服务器地址不能为空")
		return &EmailQueryOutput{
			Success: false,
			Error:   "IMAP 服务器地址不能为空",
		}, nil
	}

	if input.Username == "" {
		slog.Error("邮箱账号不能为空")
		return &EmailQueryOutput{
			Success: false,
			Error:   "邮箱账号不能为空",
		}, nil
	}

	if input.Password == "" {
		slog.Error("邮箱密码不能为空")
		return &EmailQueryOutput{
			Success: false,
			Error:   "邮箱密码不能为空",
		}, nil
	}

	// 设置默认值
	mailbox := input.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	slog.Info("开始查询邮件",
		"server", input.Server,
		"username", input.Username,
		"mailbox", mailbox,
		"limit", limit,
		"unseen", input.Unseen)

	// 连接到 IMAP 服务器
	options := &imapclient.Options{
		TLSConfig: &tls.Config{
			ServerName: parseServerName(input.Server),
		},
	}

	client, err := imapclient.DialTLS(input.Server, options)
	if err != nil {
		slog.Error("连接 IMAP 服务器失败", "err", err)
		return &EmailQueryOutput{
			Success: false,
			Error:   fmt.Sprintf("连接 IMAP 服务器失败: %v", err),
		}, nil
	}
	defer client.Close()

	// 登录
	if err := client.Login(input.Username, input.Password).Wait(); err != nil {
		slog.Error("IMAP 登录失败", "err", err)
		return &EmailQueryOutput{
			Success: false,
			Error:   fmt.Sprintf("IMAP 登录失败: %v", err),
		}, nil
	}
	defer client.Logout().Wait()

	// 选择邮箱文件夹
	selectData, err := client.Select(mailbox, nil).Wait()
	if err != nil {
		slog.Error("选择邮箱文件夹失败", "err", err, "mailbox", mailbox)
		return &EmailQueryOutput{
			Success: false,
			Error:   fmt.Sprintf("选择邮箱文件夹失败: %v", err),
		}, nil
	}

	numMessages := selectData.NumMessages
	if numMessages == 0 {
		return &EmailQueryOutput{
			Success:  true,
			Messages: []EmailMessage{},
			Count:    0,
			Message:  fmt.Sprintf("邮箱 %s 中没有邮件", mailbox),
		}, nil
	}

	// 构建搜索条件
	var searchCriteria *imap.SearchCriteria
	if input.Unseen {
		searchCriteria = &imap.SearchCriteria{
			Not: []imap.SearchCriteria{
				{Flag: []imap.Flag{imap.FlagSeen}},
			},
		}
	} else {
		searchCriteria = &imap.SearchCriteria{} // 查询所有邮件
	}

	// 搜索邮件
	searchData, err := client.Search(searchCriteria, nil).Wait()
	if err != nil {
		slog.Error("搜索邮件失败", "err", err)
		return &EmailQueryOutput{
			Success: false,
			Error:   fmt.Sprintf("搜索邮件失败: %v", err),
		}, nil
	}

	if len(searchData.AllUIDs()) == 0 {
		return &EmailQueryOutput{
			Success:  true,
			Messages: []EmailMessage{},
			Count:    0,
			Message:  fmt.Sprintf("没有找到符合条件的邮件"),
		}, nil
	}

	// 限制获取的邮件数量
	uids := searchData.AllUIDs()
	if len(uids) > limit {
		// 取最新的 limit 条
		uids = uids[len(uids)-limit:]
	}

	// 创建 UID 集合
	uidSet := imap.UIDSet{}
	uidSet.AddNum(uids...)

	// 获取邮件详情
	fetchOptions := &imap.FetchOptions{
		Envelope:    true,
		Flags:       true,
		BodySection: []*imap.FetchItemBodySection{{}},
	}

	messages := []EmailMessage{}
	fetchCmd := client.Fetch(uidSet, fetchOptions)
	defer fetchCmd.Close()

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		// 收集消息数据
		buf, err := msg.Collect()
		if err != nil {
			slog.Error("收集邮件数据失败", "err", err)
			continue
		}

		emailMsg := EmailMessage{
			UID:  uint32(buf.UID),
			Seen: hasSeenFlag(buf.Flags),
		}

		// 获取信封信息
		if buf.Envelope != nil {
			emailMsg.Subject = buf.Envelope.Subject
			emailMsg.Date = buf.Envelope.Date
			if len(buf.Envelope.From) > 0 {
				emailMsg.From = formatAddress(&buf.Envelope.From[0])
			}
			if len(buf.Envelope.To) > 0 {
				emailMsg.To = formatAddress(&buf.Envelope.To[0])
			}
		}

		// 获取邮件正文
		for _, section := range buf.BodySection {
			if section.Section.Specifier == imap.PartSpecifierNone {
				emailMsg.BodyText = extractTextFromBody(string(section.Bytes))
			}
		}

		messages = append(messages, emailMsg)
	}

	if err := fetchCmd.Close(); err != nil {
		slog.Error("获取邮件详情失败", "err", err)
		return &EmailQueryOutput{
			Success: false,
			Error:   fmt.Sprintf("获取邮件详情失败: %v", err),
		}, nil
	}

	message := fmt.Sprintf("成功查询到 %d 封邮件", len(messages))
	slog.Info(message)

	return &EmailQueryOutput{
		Success:  true,
		Messages: messages,
		Count:    len(messages),
		Message:  message,
	}, nil
}

// parseServerName 从服务器地址中解析主机名
func parseServerName(server string) string {
	// 移除端口号
	if idx := strings.LastIndex(server, ":"); idx > 0 {
		return server[:idx]
	}
	return server
}

// hasSeenFlag 检查邮件是否已读
func hasSeenFlag(flags []imap.Flag) bool {
	for _, flag := range flags {
		if flag == imap.FlagSeen {
			return true
		}
	}
	return false
}

// formatAddress 格式化邮件地址
func formatAddress(addr *imap.Address) string {
	if addr == nil {
		return ""
	}
	if addr.Name != "" {
		return fmt.Sprintf("%s <%s@%s>", addr.Name, addr.Mailbox, addr.Host)
	}
	return fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host)
}

// extractTextFromBody 从邮件正文中提取文本内容
func extractTextFromBody(body string) string {
	// 使用 mail 包解析邮件
	msg, err := mail.ReadMessage(strings.NewReader(body))
	if err != nil {
		slog.Error("解析邮件失败", "err", err)
		return body
	}

	// 读取邮件正文
	bodyBytes, err := io.ReadAll(msg.Body)
	if err != nil {
		slog.Error("读取邮件正文失败", "err", err)
		return body
	}

	text := string(bodyBytes)

	// 简单清理：移除多余的空行
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}

	return strings.Join(cleaned, "\n")
}
