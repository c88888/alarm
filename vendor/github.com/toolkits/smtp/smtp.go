package smtp

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
)

type Smtp struct {
	Address  string
	Username string
	Password string
}

func New(address, username, password string) *Smtp {
	return &Smtp{
		Address:  address,
		Username: username,
		Password: password,
	}
}

func (this *Smtp) SendMail(from, tos, subject, body string, contentType ...string) error {
	if this.Address == "" {
		return fmt.Errorf("address is necessary")
	}

	hp := strings.Split(this.Address, ":")
	if len(hp) != 2 {
		return fmt.Errorf("address format error")
	}

	arr := strings.Split(tos, ";")
	count := len(arr)
	safeArr := make([]string, 0, count)
	for i := 0; i < count; i++ {
		if arr[i] == "" {
			continue
		}
		safeArr = append(safeArr, arr[i])
	}

	if len(safeArr) == 0 {
		return fmt.Errorf("tos invalid")
	}

	tos = strings.Join(safeArr, ";")

	b64 := base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")

	header := make(map[string]string)
	header["From"] = from
	header["To"] = tos
	header["Subject"] = fmt.Sprintf("=?UTF-8?B?%s?=", b64.EncodeToString([]byte(subject)))
	header["MIME-Version"] = "1.0"

	ct := "text/plain; charset=UTF-8"
	if len(contentType) > 0 && contentType[0] == "html" {
		ct = "text/html; charset=UTF-8"
	}

	header["Content-Type"] = ct
	header["Content-Transfer-Encoding"] = "base64"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + b64.EncodeToString([]byte(body))

	auth := smtp.PlainAuth("", this.Username, this.Password, hp[0])

	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         this.Address,
	}

	host := hp[0]

	conn, err := tls.Dial("tcp", this.Address, tlsconfig)
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}

	// step 1: Use Auth
	if err = client.Auth(auth); err != nil {
		return err
	}

	// step 2: add all from and to
	if err = client.Mail(from); err != nil {
		return err
	}

	for _, k := range strings.Split(tos, ";") {
		if err = client.Rcpt(k); err != nil {
			return err
		}
	}

	// Data
	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	client.Quit()
	return nil
}
