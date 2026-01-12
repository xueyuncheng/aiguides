package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/middleware"
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
	Mailbox string `json:"mailbox,omitempty" jsonschema:"邮箱文件夹名称，默认为 INBOX"`
	Limit   int    `json:"limit,omitempty" jsonschema:"返回的邮件数量限制，默认为 10，最多 50"`
	Unseen  bool   `json:"unseen,omitempty" jsonschema:"是否只查询未读邮件，默认为 false"`
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
	Success     bool           `json:"success"`
	Messages    []EmailMessage `json:"messages,omitempty"`
	Count       int            `json:"count"`
	Message     string         `json:"message,omitempty"`
	Error       string         `json:"error,omitempty"`
	ConfigURL   string         `json:"config_url,omitempty"`   // 配置页面URL
	NeedsConfig bool           `json:"needs_config,omitempty"` // 是否需要配置
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
	// 检查 context 是否已被取消
	select {
	case <-ctx.Done():
		return &EmailQueryOutput{
			Success: false,
			Error:   "操作已取消",
		}, ctx.Err()
	default:
	}

	// 使用提供的配置查询单个服务器
	return querySingleServer(ctx, input)
}

// querySingleServer 查询单个邮件服务器
func querySingleServer(ctx context.Context, input EmailQueryInput) (*EmailQueryOutput, error) {
	// 验证并标准化输入
	input, err := validateAndNormalizeInput(input)
	if err != nil {
		slog.Error(err.Error())
		return &EmailQueryOutput{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	slog.Info("开始查询邮件",
		"mailbox", input.Mailbox,
		"limit", input.Limit,
		"unseen", input.Unseen)

	var emailServerConfigs []*table.EmailServerConfig
	tx, ok := middleware.GetTx(ctx)
	if !ok {
		slog.Error("没有从 context 中找到 db")
		output := &EmailQueryOutput{
			Success: false,
			Message: "没有从 context 中找到 db",
		}
		return output, nil
	}

	if err := tx.Find(&emailServerConfigs).Error; err != nil {
		slog.Error("tx.Find() error", "err", err)
		return nil, fmt.Errorf("tx.Find() error, err = %w", err)
	}

	if len(emailServerConfigs) == 0 {
		slog.Info("没有找到邮件服务器配置")
		configURL := "http://localhost:3000/settings/email-server-configs"
		return &EmailQueryOutput{
			Success:   false,
			Message:   "没有找到邮件服务器配置",
			ConfigURL: configURL,
		}, nil
	}

	emailServerConfig := emailServerConfigs[0]

	// 连接到 IMAP 服务器
	client, err := connectToIMAP(emailServerConfig.Server, emailServerConfig.Username, emailServerConfig.Password)
	if err != nil {
		return &EmailQueryOutput{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer client.Close()

	return fetchEmailsFromClient(ctx, client, input)
}

// fetchEmailsFromClient 从 IMAP 客户端获取邮件
func fetchEmailsFromClient(ctx context.Context, client *imapclient.Client, input EmailQueryInput) (*EmailQueryOutput, error) {
	// 选择邮箱文件夹
	selectData, err := client.Select(input.Mailbox, nil).Wait()
	if err != nil {
		slog.Error("选择邮箱文件夹失败", "err", err, "mailbox", input.Mailbox)
		return &EmailQueryOutput{
			Success: false,
			Error:   fmt.Sprintf("选择邮箱文件夹失败: %v", err),
		}, nil
	}

	if selectData.NumMessages == 0 {
		slog.Info("邮箱中没有邮件", "mailbox", input.Mailbox)
		return &EmailQueryOutput{
			Success:  true,
			Messages: []EmailMessage{},
			Count:    0,
			Message:  fmt.Sprintf("邮箱 %s 中没有邮件", input.Mailbox),
		}, nil
	}

	// 搜索并获取 UID
	uids, err := searchLatestUIDs(client, input.Unseen, input.Limit)
	if err != nil {
		slog.Error("搜索邮件失败", "err", err)
		return &EmailQueryOutput{
			Success: false,
			Error:   fmt.Sprintf("搜索邮件失败: %v", err),
		}, nil
	}

	if len(uids) == 0 {
		return &EmailQueryOutput{
			Success:  true,
			Messages: []EmailMessage{},
			Count:    0,
			Message:  "没有找到符合条件的邮件",
		}, nil
	}

	// 获取邮件详情
	messages, skippedCount, err := fetchEmails(client, uids)
	if err != nil {
		slog.Error("获取邮件详情失败", "err", err)
		return &EmailQueryOutput{
			Success: false,
			Error:   fmt.Sprintf("获取邮件详情失败: %v", err),
		}, nil
	}

	var message string
	if skippedCount > 0 {
		message = fmt.Sprintf("成功查询到 %d 封邮件（%d 封邮件获取失败）", len(messages), skippedCount)
	} else {
		message = fmt.Sprintf("成功查询到 %d 封邮件", len(messages))
	}
	slog.Info(message)

	return &EmailQueryOutput{
		Success:  true,
		Messages: messages,
		Count:    len(messages),
		Message:  message,
	}, nil
}

// validateAndNormalizeInput 验证必填参数并设置默认值
func validateAndNormalizeInput(input EmailQueryInput) (EmailQueryInput, error) {
	if input.Mailbox == "" {
		input.Mailbox = "INBOX"
	}

	if input.Limit <= 0 {
		input.Limit = 10
	}
	if input.Limit > 50 {
		input.Limit = 50
	}

	return input, nil
}

// connectToIMAP 连接并登录 IMAP 服务器
func connectToIMAP(server, username, password string) (*imapclient.Client, error) {
	options := &imapclient.Options{
		TLSConfig: &tls.Config{
			ServerName: parseServerName(server),
		},
	}

	client, err := imapclient.DialTLS(server, options)
	if err != nil {
		slog.Error("imapclient.DialTLS() error", "err", err)
		return nil, fmt.Errorf("连接 IMAP 服务器失败: %w", err)
	}

	if err := client.Login(username, password).Wait(); err != nil {
		client.Close()
		slog.Error("client.Login() error", "err", err)
		return nil, fmt.Errorf("IMAP 登录失败: %w", err)
	}

	return client, nil
}

// searchLatestUIDs 搜索邮件并返回最新的 UID 列表
func searchLatestUIDs(client *imapclient.Client, unseen bool, limit int) ([]imap.UID, error) {
	var searchCriteria *imap.SearchCriteria
	if unseen {
		searchCriteria = &imap.SearchCriteria{
			Not: []imap.SearchCriteria{
				{Flag: []imap.Flag{imap.FlagSeen}},
			},
		}
	} else {
		searchCriteria = &imap.SearchCriteria{} // 查询所有邮件
	}

	searchData, err := client.UIDSearch(searchCriteria, nil).Wait()
	if err != nil {
		return nil, err
	}

	uids := searchData.AllUIDs()
	if len(uids) == 0 {
		return []imap.UID{}, nil
	}

	// 限制获取的邮件数量
	if len(uids) > limit {
		// 取最新的 limit 条
		// uids: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
		// limit: 5
		// uids: [6, 7, 8, 9, 10]
		uids = uids[len(uids)-limit:]
	}

	return uids, nil
}

// fetchEmails 根据 UID 获取邮件详情
func fetchEmails(client *imapclient.Client, uids []imap.UID) ([]EmailMessage, int, error) {
	uidSet := imap.UIDSet{}
	uidSet.AddNum(uids...)

	fetchOptions := &imap.FetchOptions{
		Envelope:    true,
		Flags:       true,
		BodySection: []*imap.FetchItemBodySection{{}},
	}

	var messages []EmailMessage
	skippedCount := 0
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
			skippedCount++
			continue
		}

		messages = append(messages, parseEmailMessage(buf))
	}

	if err := fetchCmd.Close(); err != nil {
		return messages, skippedCount, err
	}

	return messages, skippedCount, nil
}

// parseEmailMessage 解析邮件消息
func parseEmailMessage(buf *imapclient.FetchMessageBuffer) EmailMessage {
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

	return emailMsg
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
