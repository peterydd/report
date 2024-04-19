package mail

import (
	"bytes"
	"encoding/base64"
	"log"
	"mime"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Mail interface {
	Auth()
	Send(m *Message) error
}

type smtpConf struct {
	host     string
	port     string
	username string
	password string
}

type Message struct {
	from        string
	to          []string
	cc          []string
	bcc         []string
	subject     string
	body        string
	contentType string
	attachment  *Attach
}

func SetMessage(from string, to, cc, bcc []string, subject, body, contentType string, attachment *Attach) *Message {
	return &Message{
		from:        from,
		to:          to,
		cc:          cc,
		bcc:         bcc,
		subject:     subject,
		body:        body,
		contentType: contentType,
		attachment:  attachment,
	}
}

type Attach struct {
	name        string
	contentType string
	withFile    bool
}

func SetAttach(name, contentType string, withFile bool) *Attach {
	return &Attach{
		name:        name,
		contentType: contentType,
		withFile:    withFile,
	}
}

type SendMail struct {
	*smtpConf
	auth smtp.Auth
}

func NewSendMail(host, port, username, password string) *SendMail {
	return &SendMail{
		smtpConf: &smtpConf{
			host:     host,
			port:     port,
			username: username,
			password: password,
		},
	}
}

func (sm *SendMail) Auth() {
	sm.auth = smtp.PlainAuth("", sm.username, sm.password, sm.host)
}

func (sm SendMail) Send(m *Message) error {
	sm.Auth()
	buffer := bytes.NewBuffer(nil)
	boundary := "GoBoundary"
	Header := make(map[string]string)
	Header["From"] = m.from
	Header["To"] = strings.Join(m.to, ",")
	Header["Cc"] = strings.Join(m.cc, ",")
	Header["Bcc"] = strings.Join(m.bcc, ",")
	Header["Subject"] = m.subject
	Header["Content-Type"] = "multipart/mixed;boundary=" + boundary
	Header["Mime-Version"] = "1.0"
	Header["Date"] = time.Now().String()
	writeHeader(buffer, Header)

	body := "\r\n--" + boundary + "\r\n"
	body += "Content-Type:" + m.contentType + "\r\n"
	body += "\r\n" + m.body + "\r\n"
	buffer.WriteString(body)

	if m.attachment.withFile {
		attachment := "\r\n--" + boundary + "\r\n"
		attachment += "Content-Transfer-Encoding:base64\r\n"
		attachment += "Content-Disposition:attachment\r\n"
		attachment += "Content-Type:" + m.attachment.contentType + ";name=\"" + mime.BEncoding.Encode("UTF-8", filepath.Base(m.attachment.name)) + "\"\r\n"
		buffer.WriteString(attachment)
		defer func() {
			if err := recover(); err != nil {
				log.Fatalln(err)
			}
		}()
		writeFile(buffer, m.attachment.name)
	}

	buffer.WriteString("\r\n--" + boundary + "--")

	var recipients string
	recipients = strings.Join(m.to, ",")

	if m.cc != nil {
		recipients = recipients + "," + strings.Join(m.cc, ",")
	}

	if m.bcc != nil {
		recipients = recipients + "," + strings.Join(m.bcc, ",")
	}

	rcpt := strings.Split(recipients, ",")

	if err := smtp.SendMail(sm.host+":"+sm.port, sm.auth, m.from, rcpt, buffer.Bytes()); err != nil {
		log.Fatal(err)
	}
	return nil
}

func writeHeader(buffer *bytes.Buffer, Header map[string]string) string {
	header := ""
	for key, value := range Header {
		header += key + ":" + value + "\r\n"
	}
	header += "\r\n"
	buffer.WriteString(header)
	return header
}

// read and write the file to buffer
func writeFile(buffer *bytes.Buffer, fileName string) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		panic(err.Error())
	}
	payload := make([]byte, base64.StdEncoding.EncodedLen(len(file)))
	base64.StdEncoding.Encode(payload, file)
	buffer.WriteString("\r\n")
	for index, line := 0, len(payload); index < line; index++ {
		buffer.WriteByte(payload[index])
		if (index+1)%76 == 0 {
			buffer.WriteString("\r\n")
		}
	}
}
