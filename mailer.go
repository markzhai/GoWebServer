// General smtp mailer utilities
package main

import (
	"fmt"
	"net/smtp"
	"time"
)

const (
	emailConfirmLink = "%v/u/#/pages/info?token=%v"
	emailForgetLink = "%v/u/#/pages/recover?token=%v"
	emailInvestorLink = "%v/u/#/investors/dashboard"
	emailShareholderLink = "%v/u/#/shareholders/dashboard"
	tokenExpiration = 30 * time.Minute
)

const tmpl = `From: "%v" <%v>
To: "%v" <%v>
Subject: %v
Mime-Version: 1.0
Content-Type: text/html;
 charset=UTF-8
Content-Transfer-Encoding: 7bit

%v
`

// sendMail sends out support email according to template
// and returns true on success
func sendMail(author, recipient, name, subject, message string) bool {
	auth := smtp.PlainAuth("", supportEmailUsername, supportEmailPassword,
		supportEmailHostname)
	host := fmt.Sprintf("%v:%v", supportEmailHostname, supportEmailPort)
	msg := fmt.Sprintf(tmpl, author, supportEmail, name, recipient,
		subject, message)
	err := smtp.SendMail(host, auth, supportEmail, []string{recipient},
		[]byte(msg))
	serverLog.Printf("[MXMAIL] Sent email to %v (%v): %v\n", recipient,
		subject, err)
	return err == nil
}
