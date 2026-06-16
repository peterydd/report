/*
Package mail - Mock email implementation for testing.
包 mail - 用于测试的模拟邮件实现。

This file provides MockMail for unit testing without sending real emails.
本文件提供 MockMail，用于在不发送真实邮件的情况下进行单元测试。
*/
package mail

import "fmt"

// MockMail implements the Mail interface for testing purposes.
// It records sent messages and allows simulating errors.
// MockMail 实现 Mail 接口，用于测试目的。
// 它记录已发送的邮件并允许模拟错误。
type MockMail struct {
	sentMessages []*MockMessage
	shouldError  bool
	errorMessage string
	sendCalled   int
}

// MockMessage represents a captured email message for testing.
// MockMessage 表示用于测试的已捕获邮件消息。
type MockMessage struct {
	From       string
	To         []string
	Subject    string
	Body       string
	Attachment string
}

// NewMockMail creates a new MockMail instance.
func NewMockMail() *MockMail {
	return &MockMail{
		sentMessages: make([]*MockMessage, 0),
	}
}

// Auth simulates SMTP authentication (no-op for mock).
func (m *MockMail) Auth() {
}

// Send simulates sending an email and records the message.
func (m *MockMail) Send(message *Message) error {
	m.sendCalled++

	if m.shouldError {
		return fmt.Errorf("mock send error: %s", m.errorMessage)
	}

	m.sentMessages = append(m.sentMessages, &MockMessage{
		From:    message.from,
		To:      message.to,
		Subject: message.subject,
		Body:    message.body,
	})

	return nil
}

// SetError simulates an SMTP error condition.
func (m *MockMail) SetError(message string) {
	m.shouldError = true
	m.errorMessage = message
}

// ClearError clears the simulated error condition.
func (m *MockMail) ClearError() {
	m.shouldError = false
	m.errorMessage = ""
}

// GetSendCount returns the number of times Send was called.
func (m *MockMail) GetSendCount() int {
	return m.sendCalled
}

// GetSentMessages returns all captured messages.
func (m *MockMail) GetSentMessages() []*MockMessage {
	return m.sentMessages
}

// GetLastMessage returns the most recently sent message.
func (m *MockMail) GetLastMessage() *MockMessage {
	if len(m.sentMessages) == 0 {
		return nil
	}
	return m.sentMessages[len(m.sentMessages)-1]
}
