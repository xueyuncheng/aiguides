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
	"gorm.io/gorm"
)

// EmailQueryInput 定义邮件查询工具的输入参数
type EmailQueryInput struct {
	Server   string `json:"server,omitempty" jsonschema:"IMAP 服务器地址，例如: imap.gmail.com:993。如果未提供，将使用用户配置的邮件服务器"`
	Username string `json:"username,omitempty" jsonschema:"邮箱账号。如果未提供，将使用用户配置的邮件服务器"`
	Password string `json:"password,omitempty" jsonschema:"邮箱密码或应用专用密码。如果未提供，将使用用户配置的邮件服务器"`
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
	Success        bool           `json:"success"`
	Messages       []EmailMessage `json:"messages,omitempty"`
	Count          int            `json:"count"`
	Message        string         `json:"message,omitempty"`
	Error          string         `json:"error,omitempty"`
	ConfigURL      string         `json:"config_url,omitempty"`       // 配置页面URL
	NeedsConfig    bool           `json:"needs_config,omitempty"`     // 是否需要配置
	ServerName     string         `json:"server_name,omitempty"`      // 邮件服务器名称
	TotalServers   int            `json:"total_servers,omitempty"`    // 查询的服务器总数
	SuccessServers int            `json:"success_servers,omitempty"` // 成功查询的服务器数
}

// EmailServerConfig 邮件服务器配置（内部使用）
type EmailServerConfig struct {
	ID        uint
	UserID    uint
	Server    string
	Username  string
	Password  string
	Mailbox   string
	Name      string
	IsDefault bool
}

// NewEmailQueryTool 创建邮件查询工具
//
// 该工具使用 IMAP 协议连接邮件服务器并查询邮件
func NewEmailQueryTool(db *gorm.DB, frontendURL string) (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "query_emails",
		Description: "查询邮箱中的邮件。如果用户已配置邮件服务器，将自动使用配置的服务器查询；否则需要提供服务器地址、用户名和密码。支持通过 IMAP 协议连接邮件服务器，查询指定邮箱文件夹中的邮件列表。可以查询所有邮件或仅查询未读邮件，并可限制返回的邮件数量。",
	}

	handler := func(ctx tool.Context, input EmailQueryInput) (*EmailQueryOutput, error) {
		return queryEmails(ctx, input, db, frontendURL)
	}

	return functiontool.New(config, handler)
}

// queryEmails 查询邮件
func queryEmails(ctx context.Context, input EmailQueryInput, db *gorm.DB, frontendURL string) (*EmailQueryOutput, error) {
	// 检查 context 是否已被取消
	select {
	case <-ctx.Done():
		return &EmailQueryOutput{
			Success: false,
			Error:   "操作已取消",
		}, ctx.Err()
	default:
	}

	// 如果没有提供服务器配置，尝试从数据库获取
	if input.Server == "" || input.Username == "" || input.Password == "" {
		return queryEmailsFromConfig(ctx, input, db, frontendURL)
	}

	// 使用提供的配置查询单个服务器
	return querySingleServer(ctx, input)
}

// queryEmailsFromConfig 从数据库配置查询邮件
func queryEmailsFromConfig(ctx context.Context, input EmailQueryInput, db *gorm.DB, frontendURL string) (*EmailQueryOutput, error) {
	// 从 context 获取用户ID (需要从 tool.Context 转换)
	// 注意：这里需要从请求上下文中获取用户ID，暂时返回需要配置的提示
	// TODO: 实现从 context 获取用户ID的机制
	
	var configs []EmailServerConfig
	// 暂时无法获取用户ID，返回配置提示
	// if userID, ok := ctx.Value("user_id").(uint); ok {
	//     if err := db.Where("user_id = ?", userID).Find(&configs).Error; err != nil {
	//         ...
	//     }
	// }
	
	configURL := frontendURL + "/settings/email-servers"
	
	if len(configs) == 0 {
		return &EmailQueryOutput{
			Success:     false,
			NeedsConfig: true,
			ConfigURL:   configURL,
			Error:       "邮件服务器配置未找到",
			Message:     fmt.Sprintf("您尚未配置邮件服务器。请前往 %s 配置您的邮件服务器信息（服务器地址、用户名、密码）后重试。", configURL),
		}, nil
	}

	// 查询所有配置的邮件服务器
	return queryMultipleServers(ctx, input, configs)
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
		"server", input.Server,
		"username", input.Username,
		"mailbox", input.Mailbox,
		"limit", input.Limit,
		"unseen", input.Unseen)

	// 连接到 IMAP 服务器
	client, err := connectToIMAP(input.Server, input.Username, input.Password)
	if err != nil {
		slog.Error("连接或登录 IMAP 服务器失败", "err", err)
		return &EmailQueryOutput{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer client.Close()

	return fetchEmailsFromClient(ctx, client, input)
}

// queryMultipleServers 查询多个邮件服务器
func queryMultipleServers(ctx context.Context, input EmailQueryInput, configs []EmailServerConfig) (*EmailQueryOutput, error) {
	var allMessages []EmailMessage
	successCount := 0
	totalCount := len(configs)

	for _, config := range configs {
		serverInput := input
		serverInput.Server = config.Server
		serverInput.Username = config.Username
		serverInput.Password = config.Password
		if serverInput.Mailbox == "" && config.Mailbox != "" {
			serverInput.Mailbox = config.Mailbox
		}

		result, err := querySingleServer(ctx, serverInput)
		if err != nil || !result.Success {
			slog.Warn("查询邮件服务器失败",
				"server", config.Name,
				"error", result.Error)
			continue
		}

		successCount++
		allMessages = append(allMessages, result.Messages...)
	}

	if successCount == 0 {
		return &EmailQueryOutput{
			Success:        false,
			TotalServers:   totalCount,
			SuccessServers: 0,
			Error:          "所有邮件服务器查询失败",
		}, nil
	}

	return &EmailQueryOutput{
		Success:        true,
		Messages:       allMessages,
		Count:          len(allMessages),
		TotalServers:   totalCount,
		SuccessServers: successCount,
		Message:        fmt.Sprintf("成功从 %d/%d 个邮件服务器查询到 %d 封邮件", successCount, totalCount, len(allMessages)),
	}, nil
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
	// Server, Username, Password 现在是可选的（可以从数据库配置获取）
	if input.Server == "" {
		return input, fmt.Errorf("IMAP 服务器地址不能为空")
	}
	if input.Username == "" {
		return input, fmt.Errorf("邮箱账号不能为空")
	}
	if input.Password == "" {
		return input, fmt.Errorf("邮箱密码不能为空")
	}

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
		return nil, fmt.Errorf("连接 IMAP 服务器失败: %v", err)
	}

	if err := client.Login(username, password).Wait(); err != nil {
		client.Close()
		return nil, fmt.Errorf("IMAP 登录失败: %v", err)
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
