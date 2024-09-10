package notifications

var (
	GetSmtpInfo = getSmtpInfo
)

var ExtractSmtpConfiguration = func(info *smtpInfo) (string, string, string, string, string, string, string) {
	return info.recipients, info.bcc, info.identity, info.fromEmail, info.password, info.host, info.port
}
