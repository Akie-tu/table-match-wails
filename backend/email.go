package backend

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
)

// 邮箱配置
type EmailConfig struct {
	SenderEmail string `json:"sender_email"` // 发件邮箱
	AuthCode    string `json:"auth_code"`    // SMTP授权码
	SMTPHost    string `json:"smtp_host"`    // smtp.qq.com
	SMTPPort    string `json:"smtp_port"`    // 465/587
	SenderName  string `json:"sender_name"`  // 发件人显示名(纯ASCII)
}

// 发件结果
type EmailResult struct {
	OK   bool   `json:"ok"`
	Msg  string `json:"msg"`
}

// 配置文件路径: 程序当前目录(可写时), 否则用户配置目录(与固定内容同目录)
func emailConfigPath() string {
	return filepath.Join(appConfigDir(), "email_config.json")
}

// 保存配置
func SaveEmailConfig(cfg *EmailConfig) error {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(emailConfigPath(), data, 0o600)
}

// 读取配置
func LoadEmailConfig() (*EmailConfig, error) {
	data, err := os.ReadFile(emailConfigPath())
	if err != nil {
		return nil, err
	}
	var cfg EmailConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// 发送邮件(带附件)
func SendEmail(cfg *EmailConfig, to, subject, body string, attachments []string) (*EmailResult, error) {
	if cfg == nil || cfg.SenderEmail == "" || cfg.AuthCode == "" {
		return &EmailResult{OK: false, Msg: "未配置邮箱, 请先设置"}, nil
	}
	if strings.TrimSpace(to) == "" {
		return &EmailResult{OK: false, Msg: "收件人为空"}, nil
	}

	// 构造MIME邮件(支持附件)
	var msg strings.Builder
	boundary := "----tablematch-boundary-7f4k2"
	name := cfg.SenderName
	if name == "" {
		name = cfg.SenderEmail
	}
	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", name, cfg.SenderEmail))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: =?UTF-8?B?%s?=\r\n", base64Encode(subject)))
	msg.WriteString("MIME-Version: 1.0\r\n")
	if len(attachments) > 0 {
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary))
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		msg.WriteString(base64Encode(body) + "\r\n")
		for _, ap := range attachments {
			data, err := os.ReadFile(ap)
			if err != nil {
				return &EmailResult{OK: false, Msg: fmt.Sprintf("附件读取失败: %s", filepath.Base(ap))}, nil
			}
			fname := filepath.Base(ap)
			msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			msg.WriteString("Content-Type: application/octet-stream; name=\"=?UTF-8?B?" +
				base64Encode(fname) + "?=\"\r\n")
			msg.WriteString("Content-Transfer-Encoding: base64\r\n")
			msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"=?UTF-8?B?%s?=\"\r\n\r\n", base64Encode(fname)))
			msg.WriteString(base64Encode(string(data)) + "\r\n")
			msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
		}
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		msg.WriteString(base64Encode(body))
	}

	// SMTP发送
	host := cfg.SMTPHost
	port := cfg.SMTPPort
	if host == "" {
		host = "smtp.qq.com"
	}
	if port == "" {
		port = "465"
	}
	addr := fmt.Sprintf("%s:%s", host, port)

	var err error
	if port == "465" {
		err = sendSSL(addr, cfg, to, msg.String())
	} else {
		err = sendStartTLS(addr, cfg, to, msg.String())
	}
	if err != nil {
		return &EmailResult{OK: false, Msg: fmt.Sprintf("发送失败: %v", err)}, nil
	}
	return &EmailResult{OK: true, Msg: "发送成功"}, nil
}

// 邮箱预设结果(单对象返回, 避免多返回值JS问题)
type PresetResult struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

// QQ/163预设
func EmailPreset(provider string) PresetResult {
	switch provider {
	case "163":
		return PresetResult{Host: "smtp.163.com", Port: "465"}
	default:
		return PresetResult{Host: "smtp.qq.com", Port: "465"}
	}
}

// base64编码(MIME用)
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// SSL直接拨号(465)
func sendSSL(addr string, cfg *EmailConfig, to, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: strings.Split(addr, ":")[0], InsecureSkipVerify: false})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return err
	}
	return doSend(c, cfg, to, msg)
}

// STARTTLS(587)
func sendStartTLS(addr string, cfg *EmailConfig, to, msg string) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: strings.Split(addr, ":")[0]}); err != nil {
			return err
		}
	}
	return doSend(c, cfg, to, msg)
}

func doSend(c *smtp.Client, cfg *EmailConfig, to, msg string) error {
	defer c.Close()
	if ok, _ := c.Extension("AUTH"); ok {
		auth := smtp.PlainAuth("", cfg.SenderEmail, cfg.AuthCode, strings.Split(cfg.SMTPHost, ":")[0])
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("认证失败: %v", err)
		}
	}
	if err := c.Mail(cfg.SenderEmail); err != nil {
		return err
	}
	if err := c.Rcpt(strings.Split(to, ",")[0]); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	return w.Close()
}