# Email Query Tool - Usage Examples

This document provides examples of how to interact with the AI assistant to use the email query tool.

## Example 1: Query Latest Emails

**User:**
```
请查询我的邮箱中最新的邮件
```

**AI Assistant Response:**
```
好的，我需要以下信息来查询您的邮箱：

1. IMAP 服务器地址（例如：imap.gmail.com:993）
2. 邮箱账号
3. 邮箱密码或应用专用密码

请提供这些信息。
```

**User:**
```
服务器: imap.gmail.com:993
账号: myemail@gmail.com
密码: [应用专用密码]
```

**AI Assistant Action:**
The AI will call the `query_emails` tool with these parameters and return the email list.

---

## Example 2: Query Unread Emails

**User:**
```
帮我查看有哪些未读邮件
```

**AI Assistant:**
```
我将为您查询未读邮件，请提供：
- IMAP 服务器地址
- 邮箱账号
- 密码
```

The AI will set `unseen=true` in the tool parameters to filter only unread emails.

---

## Example 3: Query Specific Number of Emails

**User:**
```
查询最近 20 封邮件
```

**AI Assistant:**
The AI will set `limit=20` to retrieve 20 emails.

---

## Example 4: Query Specific Mailbox Folder

**User:**
```
查询我的已发送邮件
```

**AI Assistant:**
The AI will set `mailbox="Sent"` to query the Sent folder instead of INBOX.

---

## Tool Parameters

The `query_emails` tool accepts the following parameters:

```json
{
  "server": "imap.gmail.com:993",
  "username": "user@example.com",
  "password": "app_password",
  "mailbox": "INBOX",
  "limit": 10,
  "unseen": false
}
```

## Tool Response Format

```json
{
  "success": true,
  "count": 5,
  "message": "成功查询到 5 封邮件",
  "messages": [
    {
      "uid": 12345,
      "subject": "Meeting Tomorrow",
      "from": "John Doe <john@example.com>",
      "to": "user@example.com",
      "date": "2026-01-10T10:00:00Z",
      "seen": false,
      "body_text": "Hi, just a reminder about our meeting tomorrow..."
    }
  ]
}
```

## Security Best Practices

1. **Use App-Specific Passwords**: For Gmail and other services that support 2FA, always use app-specific passwords
2. **Don't Share Credentials**: The AI assistant doesn't store your credentials, but be cautious about sharing them
3. **Secure Connection**: All connections use TLS encryption
4. **Monitor Access**: Regularly review your email account's security settings and access logs

## Supported Email Providers

| Provider | IMAP Server | Port | Notes |
|----------|-------------|------|-------|
| Gmail | imap.gmail.com | 993 | Requires app-specific password |
| Outlook/Hotmail | outlook.office365.com | 993 | Standard account password |
| QQ Mail | imap.qq.com | 993 | Need to enable IMAP in settings |
| 163 Mail | imap.163.com | 993 | Need to enable IMAP in settings |
| Yahoo Mail | imap.mail.yahoo.com | 993 | Requires app-specific password |

## Troubleshooting

### Common Issues

1. **Authentication Failed**
   - Verify username and password are correct
   - For Gmail: Use app-specific password, not your regular password
   - Check if 2FA is enabled and configured properly

2. **Connection Timeout**
   - Verify the server address and port
   - Check if IMAP is enabled in your email settings
   - Ensure your firewall allows outbound connections on port 993

3. **No Emails Returned**
   - Check if the mailbox name is correct (case-sensitive)
   - Verify the mailbox contains emails
   - Try with `unseen=false` to query all emails

## Integration with AI Assistant

The email query tool is automatically integrated into the SearchAgent. The AI assistant can:

- Understand natural language requests about email queries
- Ask for required information if not provided
- Parse and present email information in a readable format
- Handle errors gracefully and provide helpful error messages
- Suggest solutions for common issues
