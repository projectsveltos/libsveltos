package notifications

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	webexteams "github.com/jbogarin/go-cisco-webex-teams/sdk"
)

var (
	GetSmtpInfo  = getSmtpInfo
	GetWebexInfo = getWebexInfo
)

var ExtractSmtpConfiguration = func(info *smtpInfo) (string, string, string, string, string, string, string) {
	return info.recipients, info.bcc, info.identity, info.fromEmail, info.password, info.host, info.port
}

var ExtractWebexConfiguration = func(info *webexInfo) (string, string) {
	return info.room, info.token
}

var MockCreateMessage = func(err bool) {
	sendWebexMessage = func(wc *webexteams.Client, message *webexteams.MessageCreateRequest) (*webexteams.Message, *resty.Response, error) {
		if err {
			return nil, nil, fmt.Errorf("failed to send message")
		}
		return nil, nil, nil
	}
}
