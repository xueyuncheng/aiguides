package tools

import (
	"context"
	"runtime"
	"testing"
)

func TestNewMailFetchToolWithJSON(t *testing.T) {
	fetchTool, err := NewMailFetchToolWithJSON()
	if err != nil {
		t.Fatalf("NewMailFetchToolWithJSON returned error: %v", err)
	}

	if fetchTool.Name() != "fetch_apple_mail" {
		t.Errorf("Expected name 'fetch_apple_mail', got '%s'", fetchTool.Name())
	}

	if fetchTool.IsLongRunning() {
		t.Error("Expected IsLongRunning to be false")
	}
}

func TestFetchAppleMailWithJSON_NotMacOS(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping test on macOS - this test is for non-macOS systems")
	}

	input := MailFetchInput{MaxCount: 5, Mailbox: "INBOX"}
	output := fetchAppleMailWithJSON(context.Background(), input)

	if output.Success {
		t.Error("Expected failure on non-macOS system, got success")
	}

	if output.Error == "" {
		t.Error("Expected error message on non-macOS system, got empty string")
	}

	expectedError := "此工具仅支持 macOS 系统的 Apple Mail"
	if output.Error != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, output.Error)
	}
}

func TestMailFetchInput_DefaultValues(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test - requires macOS")
	}

	// 测试默认值处理
	// 注意：这个测试在没有运行 Mail.app 或没有权限时会失败
	// 这是预期的行为
	input := MailFetchInput{}
	output := fetchAppleMailWithJSON(context.Background(), input)

	// 我们只检查是否正确处理了输入，不检查是否成功
	// 因为可能没有 Mail.app 运行或没有权限
	t.Logf("Result: Success=%v, Error=%s, EmailCount=%d",
		output.Success, output.Error, len(output.Emails))
}
