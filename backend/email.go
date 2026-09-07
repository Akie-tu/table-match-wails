package backend

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	OK  bool   `json:"ok"`
	Msg string `json:"msg"`
}

// 附件大小上限: 25MB(留余量, QQ邮箱普通附件上限50MB)
const maxAttachmentSize = 25 << 20

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

// 附件安全校验: 存在/大小/文件名清洗(防MIME头注入), 返回(清洗后文件名, 错误)
func validateAttachment(ap string) (string, error) {
	fi, err := os.Stat(ap)
	if err != nil {
		return "", fmt.Errorf("附件不存在: %s", ap)
	}
	if fi.Size() > maxAttachmentSize {
		return "", fmt.Errorf("附件过大(>%dMB): %s", maxAttachmentSize>>20, filepath.Base(ap))
	}
	name := filepath.Base(ap)
	// 清洗: 去引号/换行/控制字符, 防破坏MIME头
	name = strings.Map(func(r rune) rune {
		if r == '"' || r == '\r' || r == '\n' || r < 0x20 {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("附件名非法: %s", filepath.Base(ap))
	}
	return name, nil
}

// 发送邮件(带附件, 支持任意文件类型)
func SendEmail(cfg *EmailConfig, to, subject, body string, attachments []string) (*EmailResult, error) {
	if cfg == nil || cfg.SenderEmail == "" || cfg.AuthCode == "" {
		return &EmailResult{OK: false, Msg: "未配置邮箱, 请先设置"}, nil
	}
	if strings.TrimSpace(to) == "" {
		return &EmailResult{OK: false, Msg: "收件人为空"}, nil
	}

	// 构造MIME邮件
	var msg strings.Builder
	boundary := "----tablematch-boundary-7f4k2"
	name := cfg.SenderName
	if name == "" {
		name = cfg.SenderEmail
	}
	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", name, cfg.SenderEmail))
	// To头保留用户原始写法(显示用), 实际投递以Rcpt为准
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: =?UTF-8?B?%s?=\r\n", base64Encode(subject)))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if len(attachments) > 0 {
		// 先校验全部附件, 任一失败则整体不发(避免发一半)
		type att struct {
			path, name string
			data       []byte
		}
		atts := make([]att, 0, len(attachments))
		for _, ap := range attachments {
			aname, err := validateAttachment(ap)
			if err != nil {
				return &EmailResult{OK: false, Msg: err.Error()}, nil
			}
			data, err := os.ReadFile(ap)
			if err != nil {
				return &EmailResult{OK: false, Msg: fmt.Sprintf("附件读取失败: %s", aname)}, nil
			}
			atts = append(atts, att{path: ap, name: aname, data: data})
		}

		msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary))
		// 正文part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		msg.WriteString(wrap76(base64Encode(body)) + "\r\n")
		// 附件parts: 任意类型统一 octet-stream, 接收端按扩展名识别
		for _, a := range atts {
			encName := mime.QEncoding.Encode("UTF-8", a.name)
			msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			msg.WriteString(fmt.Sprintf("Content-Type: application/octet-stream; name=\"%s\"\r\n", encName))
			msg.WriteString("Content-Transfer-Encoding: base64\r\n")
			msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", encName))
			msg.WriteString(wrap76(base64.StdEncoding.EncodeToString(a.data)) + "\r\n")
		}
		// 结束边界: 整个multipart只写一次(修复: 原来写在循环内导致多附件结构损坏)
		msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		msg.WriteString(wrap76(base64Encode(body)))
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

// RFC 2045: base64每76字符插入CRLF折行
func wrap76(s string) string {
	if len(s) <= 76 {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i += 76 {
		e := i + 76
		if e > len(s) {
			e = len(s)
		}
		b.WriteString(s[i:e])
		if e < len(s) {
			b.WriteString("\r\n")
		}
	}
	return b.String()
}

// 拆分收件人: 支持半角/全角逗号、分号、空格分隔
func splitRecipients(to string) []string {
	fields := strings.FieldsFunc(to, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

// SSL直接拨号(465)
func sendSSL(addr string, cfg *EmailConfig, to, msg string) error {
	d := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := tls.DialWithDialer(d, "tcp", addr, &tls.Config{ServerName: strings.Split(addr, ":")[0], InsecureSkipVerify: false})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		conn.Close()
		return err
	}
	return doSend(c, cfg, to, msg)
}

// STARTTLS(587)
func sendStartTLS(addr string, cfg *EmailConfig, to, msg string) error {
	d := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		conn.Close()
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: strings.Split(addr, ":")[0]}); err != nil {
			c.Close()
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
	// 多收件人: 逐个Rcpt, 任一被拒则整体报错(修复: 原来只发第一个)
	rcpts := splitRecipients(to)
	if len(rcpts) == 0 {
		return fmt.Errorf("收件人解析为空")
	}
	var rejected []string
	for _, r := range rcpts {
		if err := c.Rcpt(r); err != nil {
			rejected = append(rejected, fmt.Sprintf("%s(%v)", r, err))
		}
	}
	if len(rejected) > 0 {
		return fmt.Errorf("以下收件人被拒绝: %s", strings.Join(rejected, ", "))
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
