package notifications

var (
	GetSmtpInfo = getSmtpInfo
)

var ExtractSmtpConfiguration = func(info *smtpInfo) ([]string, []string, []string, string, string, string, string) {
	return info.to, info.cc, info.bcc, info.fromEmail, info.password, info.host, info.port
}
