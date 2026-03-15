package tools

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"mime"
	"net/mail"
	"net/smtp"
	"sort"
	"strings"

	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/middleware"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const emailServerConfigURL = "http://localhost:3000/settings/email-server-configs"

type SendEmailInput struct {
	ConfigName string   `json:"config_name,omitempty" jsonschema:"可选，指定使用的邮件配置名称；不填时优先使用默认配置"`
	To         []string `json:"to,omitempty" jsonschema:"收件人邮箱地址列表；留空时默认发送给发件人自己"`
	Cc         []string `json:"cc,omitempty" jsonschema:"抄送邮箱地址列表（可选）"`
	Bcc        []string `json:"bcc,omitempty" jsonschema:"密送邮箱地址列表（可选）"`
	Subject    string   `json:"subject" jsonschema:"邮件主题"`
	Body       string   `json:"body" jsonschema:"邮件正文内容"`
	IsHTML     bool     `json:"is_html,omitempty" jsonschema:"正文是否为 HTML，默认为纯文本"`
	FromName   string   `json:"from_name,omitempty" jsonschema:"可选，自定义发件人名称；不填则使用配置名称"`
}

type SendEmailOutput struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message,omitempty"`
	Error       string   `json:"error,omitempty"`
	UsedConfig  string   `json:"used_config,omitempty"`
	Recipients  []string `json:"recipients,omitempty"`
	ConfigURL   string   `json:"config_url,omitempty"`
	NeedsConfig bool     `json:"needs_config,omitempty"`
}

func NewSendEmailTool() (tool.Tool, error) {
	config := functiontool.Config{
		Name:        "send_email",
		Description: "发送邮件。使用用户已配置的 SMTP 服务器向一个或多个收件人发送纯文本或 HTML 邮件。适合草拟后正式发出通知、回复或汇报。",
	}

	handler := func(ctx tool.Context, input SendEmailInput) (*SendEmailOutput, error) {
		return sendEmail(ctx, input)
	}

	return functiontool.New(config, handler)
}

func sendEmail(ctx context.Context, input SendEmailInput) (*SendEmailOutput, error) {
	validatedInput, err := validateSendEmailInput(input)
	if err != nil {
		slog.Error(err.Error())
		return &SendEmailOutput{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	emailServerConfig, output, err := loadEmailServerConfigForSending(ctx, validatedInput.ConfigName)
	if err != nil {
		return nil, err
	}
	if output != nil {
		return output, nil
	}

	fromName := validatedInput.FromName
	if fromName == "" {
		fromName = emailServerConfig.Name
	}

	defaultedToSelf, err := populateDefaultRecipient(&validatedInput, emailServerConfig.Username)
	if err != nil {
		slog.Error("populateDefaultRecipient() error", "username", emailServerConfig.Username, "err", err)
		return &SendEmailOutput{
			Success:    false,
			Error:      fmt.Sprintf("默认收件人无效: %v", err),
			UsedConfig: emailServerConfig.Name,
		}, nil
	}

	recipients := append(append(append([]string{}, validatedInput.To...), validatedInput.Cc...), validatedInput.Bcc...)
	slog.Info("开始发送邮件",
		"config_name", emailServerConfig.Name,
		"smtp_server", emailServerConfig.SMTPServer,
		"default_to_self", defaultedToSelf,
		"to_count", len(validatedInput.To),
		"cc_count", len(validatedInput.Cc),
		"bcc_count", len(validatedInput.Bcc),
		"recipient_count", len(recipients),
		"subject", validatedInput.Subject,
	)

	message, err := buildEmailMessage(emailServerConfig.Username, fromName, validatedInput, recipients)
	if err != nil {
		slog.Error("buildEmailMessage() error", "err", err)
		return nil, fmt.Errorf("buildEmailMessage() error, err = %w", err)
	}

	if err := sendSMTPMessage(emailServerConfig.SMTPServer, emailServerConfig.Username, emailServerConfig.Password, recipients, message); err != nil {
		slog.Error("sendSMTPMessage() error", "smtp_server", emailServerConfig.SMTPServer, "config_name", emailServerConfig.Name, "err", err)
		return &SendEmailOutput{
			Success:    false,
			Error:      fmt.Sprintf("发送邮件失败: %v", err),
			UsedConfig: emailServerConfig.Name,
		}, nil
	}

	slog.Info("邮件发送成功",
		"config_name", emailServerConfig.Name,
		"smtp_server", emailServerConfig.SMTPServer,
		"default_to_self", defaultedToSelf,
		"recipient_count", len(recipients),
		"subject", validatedInput.Subject,
	)

	return &SendEmailOutput{
		Success:    true,
		Message:    fmt.Sprintf("邮件已成功发送给 %d 位收件人", len(recipients)),
		UsedConfig: emailServerConfig.Name,
		Recipients: recipients,
	}, nil
}

func validateSendEmailInput(input SendEmailInput) (SendEmailInput, error) {
	input.ConfigName = strings.TrimSpace(input.ConfigName)
	input.Subject = strings.TrimSpace(input.Subject)
	input.Body = strings.TrimSpace(input.Body)
	input.FromName = strings.TrimSpace(input.FromName)

	if input.Subject == "" {
		return SendEmailInput{}, fmt.Errorf("邮件主题不能为空")
	}
	if input.Body == "" {
		return SendEmailInput{}, fmt.Errorf("邮件正文不能为空")
	}

	var err error
	input.To, err = normalizeEmailAddresses(input.To)
	if err != nil {
		return SendEmailInput{}, fmt.Errorf("收件人地址无效: %w", err)
	}
	input.Cc, err = normalizeEmailAddresses(input.Cc)
	if err != nil {
		return SendEmailInput{}, fmt.Errorf("抄送地址无效: %w", err)
	}
	input.Bcc, err = normalizeEmailAddresses(input.Bcc)
	if err != nil {
		return SendEmailInput{}, fmt.Errorf("密送地址无效: %w", err)
	}

	return input, nil
}

func populateDefaultRecipient(input *SendEmailInput, senderAddress string) (bool, error) {
	if len(input.To) > 0 {
		return false, nil
	}

	defaultRecipients, err := normalizeEmailAddresses([]string{senderAddress})
	if err != nil {
		return false, err
	}
	if len(defaultRecipients) == 0 {
		return false, fmt.Errorf("发件人邮箱不能为空")
	}

	input.To = defaultRecipients
	return true, nil
}

func normalizeEmailAddresses(addresses []string) ([]string, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(addresses))
	seen := make(map[string]struct{}, len(addresses))
	for _, rawAddress := range addresses {
		parsedAddress, err := mail.ParseAddress(strings.TrimSpace(rawAddress))
		if err != nil {
			return nil, fmt.Errorf("%q: %w", rawAddress, err)
		}
		address := strings.ToLower(strings.TrimSpace(parsedAddress.Address))
		if _, exists := seen[address]; exists {
			continue
		}
		seen[address] = struct{}{}
		normalized = append(normalized, address)
	}

	sort.Strings(normalized)
	return normalized, nil
}

func loadEmailServerConfigForSending(ctx context.Context, configName string) (*table.EmailServerConfig, *SendEmailOutput, error) {
	tx, ok := middleware.GetTx(ctx)
	if !ok {
		slog.Error("没有从 context 中找到 db")
		return nil, &SendEmailOutput{
			Success: false,
			Error:   "没有从 context 中找到 db",
		}, nil
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		slog.Error("没有从 context 中找到 user id")
		return nil, &SendEmailOutput{
			Success: false,
			Error:   "没有从 context 中找到 user id",
		}, nil
	}

	var emailServerConfigs []*table.EmailServerConfig
	query := tx.Where("user_id = ?", userID).Order("is_default DESC, created_at DESC")
	if configName != "" {
		query = query.Where("name = ?", configName)
	}
	if err := query.Find(&emailServerConfigs).Error; err != nil {
		slog.Error("query.Find() error", "user_id", userID, "config_name", configName, "err", err)
		return nil, nil, fmt.Errorf("query.Find() error, err = %w", err)
	}

	if len(emailServerConfigs) == 0 {
		message := "没有找到邮件服务器配置"
		if configName != "" {
			message = fmt.Sprintf("没有找到名为 %q 的邮件配置", configName)
		}
		return nil, &SendEmailOutput{
			Success:     false,
			Message:     message,
			ConfigURL:   emailServerConfigURL,
			NeedsConfig: true,
		}, nil
	}

	emailServerConfig := emailServerConfigs[0]
	if strings.TrimSpace(emailServerConfig.SMTPServer) == "" {
		return nil, &SendEmailOutput{
			Success:     false,
			Message:     fmt.Sprintf("邮件配置 %q 尚未设置 SMTP 服务器，暂时无法发送邮件", emailServerConfig.Name),
			UsedConfig:  emailServerConfig.Name,
			ConfigURL:   emailServerConfigURL,
			NeedsConfig: true,
		}, nil
	}

	return emailServerConfig, nil, nil
}

func buildEmailMessage(fromAddress, fromName string, input SendEmailInput, recipients []string) ([]byte, error) {
	var buffer bytes.Buffer

	fromHeader := formatHeaderAddress(fromName, fromAddress)
	if fromHeader == "" {
		return nil, fmt.Errorf("发件人地址不能为空")
	}

	headers := []string{
		fmt.Sprintf("From: %s", fromHeader),
		fmt.Sprintf("To: %s", strings.Join(input.To, ", ")),
		fmt.Sprintf("Subject: %s", encodeHeader(input.Subject)),
		"MIME-Version: 1.0",
	}
	if len(input.Cc) > 0 {
		headers = append(headers, fmt.Sprintf("Cc: %s", strings.Join(input.Cc, ", ")))
	}
	contentType := "text/plain; charset=UTF-8"
	if input.IsHTML {
		contentType = "text/html; charset=UTF-8"
	}
	headers = append(headers,
		fmt.Sprintf("Content-Type: %s", contentType),
		"Content-Transfer-Encoding: 8bit",
	)

	for _, header := range headers {
		buffer.WriteString(header)
		buffer.WriteString("\r\n")
	}
	buffer.WriteString("\r\n")
	buffer.WriteString(normalizeCRLF(input.Body))
	if !strings.HasSuffix(buffer.String(), "\r\n") {
		buffer.WriteString("\r\n")
	}

	if len(recipients) == 0 {
		return nil, fmt.Errorf("收件人列表不能为空")
	}

	return buffer.Bytes(), nil
}

func sendSMTPMessage(serverAddr, username, password string, recipients []string, message []byte) error {
	host := parseServerName(serverAddr)
	if host == "" {
		slog.Error("SMTP 服务器地址不能为空")
		return fmt.Errorf("SMTP 服务器地址不能为空")
	}

	if strings.HasSuffix(serverAddr, ":465") {
		conn, err := tls.Dial("tcp", serverAddr, &tls.Config{ServerName: host})
		if err != nil {
			slog.Error("tls.Dial() error", "smtp_server", serverAddr, "err", err)
			return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
		}

		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			slog.Error("smtp.NewClient() error", "smtp_server", serverAddr, "err", err)
			return fmt.Errorf("初始化 SMTP 客户端失败: %w", err)
		}

		return sendSMTPWithClient(client, host, username, password, recipients, message)
	}

	client, err := smtp.Dial(serverAddr)
	if err != nil {
		slog.Error("smtp.Dial() error", "smtp_server", serverAddr, "err", err)
		return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			client.Close()
			slog.Error("client.StartTLS() error", "smtp_server", serverAddr, "err", err)
			return fmt.Errorf("启动 SMTP TLS 失败: %w", err)
		}
	}

	return sendSMTPWithClient(client, host, username, password, recipients, message)
}

func sendSMTPWithClient(client *smtp.Client, host, username, password string, recipients []string, message []byte) error {
	defer client.Close()

	if ok, _ := client.Extension("AUTH"); ok && username != "" {
		if err := client.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
			slog.Error("client.Auth() error", "smtp_host", host, "username", username, "err", err)
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
	}

	if err := client.Mail(username); err != nil {
		slog.Error("client.Mail() error", "from", username, "err", err)
		return fmt.Errorf("设置发件人失败: %w", err)
	}

	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			slog.Error("client.Rcpt() error", "recipient", recipient, "err", err)
			return fmt.Errorf("添加收件人 %s 失败: %w", recipient, err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		slog.Error("client.Data() error", "err", err)
		return fmt.Errorf("创建邮件数据流失败: %w", err)
	}

	if _, err := writer.Write(message); err != nil {
		writer.Close()
		slog.Error("writer.Write() error", "err", err)
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		slog.Error("writer.Close() error", "err", err)
		return fmt.Errorf("完成邮件写入失败: %w", err)
	}

	if err := client.Quit(); err != nil {
		slog.Error("client.Quit() error", "err", err)
		return fmt.Errorf("结束 SMTP 会话失败: %w", err)
	}

	return nil
}

func formatHeaderAddress(name, address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return ""
	}
	if name == "" {
		return address
	}
	if isASCII(name) {
		return (&mail.Address{Name: name, Address: address}).String()
	}
	return fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("UTF-8", name), address)
}

func encodeHeader(value string) string {
	if isASCII(value) {
		return value
	}
	return mime.QEncoding.Encode("UTF-8", value)
}

func isASCII(value string) bool {
	for _, r := range value {
		if r > 127 {
			return false
		}
	}
	return true
}

func normalizeCRLF(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	return strings.ReplaceAll(body, "\n", "\r\n")
}
