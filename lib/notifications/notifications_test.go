package notifications_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"

	smtpmock "github.com/mocktools/go-smtp-mock/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	"github.com/projectsveltos/libsveltos/lib/notifications"
)

var _ = Describe("Notification", func() {
	It("getSmtpInfo get smtp information from Secret", func() {
		smtpRecipients := fmt.Sprintf("%s@a.com,%s@b.com", randomString(), randomString())
		smtpBcc := fmt.Sprintf("%s@c.com", randomString())
		smtpIdentity := randomString()
		smtpSender := fmt.Sprintf("%s@d.com", randomString())
		smtpPassword := randomString()
		smtpHost := fmt.Sprintf("%s.com", randomString())
		smtpPort := rand.IntnRange(444, 9999)

		data := map[string][]byte{
			libsveltosv1beta1.SmtpRecipients: []byte(smtpRecipients),
			libsveltosv1beta1.SmtpBcc:        []byte(smtpBcc),
			libsveltosv1beta1.SmtpIdentity:   []byte(smtpIdentity),
			libsveltosv1beta1.SmtpSender:     []byte(smtpSender),
			libsveltosv1beta1.SmtpPassword:   []byte(smtpPassword),
			libsveltosv1beta1.SmtpHost:       []byte(smtpHost),
			libsveltosv1beta1.SmtpPort:       []byte(fmt.Sprintf("%d", smtpPort)),
		}

		ns, sec := createNamespaceAndSecret(data)

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  ns,
				Name:       sec,
			},
		}

		smptInfo, err := notifications.GetSmtpInfo(context.TODO(), k8sClient, notification)
		Expect(err).To(BeNil())
		Expect(smptInfo).ToNot(BeNil())

		recip, bcc, identity, sender, pass, host, port := notifications.ExtractSmtpConfiguration(smptInfo)
		Expect(recip).To(Equal(smtpRecipients))
		Expect(bcc).To(Equal(smtpBcc))
		Expect(identity).To(Equal(smtpIdentity))
		Expect(sender).To(Equal(smtpSender))
		Expect(pass).To(Equal(smtpPassword))
		Expect(host).To(Equal(smtpHost))
		Expect(port).To(Equal(fmt.Sprintf("%d", smtpPort)))
	})
	It("getSmtpInfo raises exception if Secret Data is nil", func() {
		data := map[string][]byte{}
		ns, sec := createNamespaceAndSecret(data)

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  ns,
				Name:       sec,
			},
		}

		smptInfo, err := notifications.GetSmtpInfo(context.TODO(), k8sClient, notification)
		Expect(smptInfo).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err).To(Equal(fmt.Errorf("notification must reference v1 secret containing notification configuration")))
	})
	It("getSmtpInfo raises exception if NotificationRef is nil", func() {
		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
		}

		_, err := notifications.GetSmtpInfo(context.TODO(), k8sClient, notification)
		Expect(err).ToNot(BeNil())
		Expect(err).To(Equal(fmt.Errorf("notification must reference v1 secret containing notification configuration")))
	})
	It("NewMailer creates a new mailer", func() {
		smtpRecipients := fmt.Sprintf("%s@a.com,%s@b.com", randomString(), randomString())
		smtpBcc := fmt.Sprintf("%s@c.com", randomString())
		smtpIdentity := randomString()
		smtpSender := fmt.Sprintf("%s@d.com", randomString())
		smtpPassword := randomString()
		smtpHost := fmt.Sprintf("%s.com", randomString())
		smtpPort := rand.IntnRange(444, 9999)

		data := map[string][]byte{
			libsveltosv1beta1.SmtpRecipients: []byte(smtpRecipients),
			libsveltosv1beta1.SmtpBcc:        []byte(smtpBcc),
			libsveltosv1beta1.SmtpIdentity:   []byte(smtpIdentity),
			libsveltosv1beta1.SmtpSender:     []byte(smtpSender),
			libsveltosv1beta1.SmtpPassword:   []byte(smtpPassword),
			libsveltosv1beta1.SmtpHost:       []byte(smtpHost),
			libsveltosv1beta1.SmtpPort:       []byte(fmt.Sprintf("%d", smtpPort)),
		}

		ns, sec := createNamespaceAndSecret(data)

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  ns,
				Name:       sec,
			},
		}

		mailer, err := notifications.NewMailer(context.Background(), k8sClient, notification)
		Expect(err).To(BeNil())
		Expect(mailer).ToNot(BeNil())
	})
	It("NewMailer raises exception if notification is nil", func() {
		mailer, err := notifications.NewMailer(context.Background(), k8sClient, &libsveltosv1beta1.Notification{})
		Expect(mailer).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err).To(Equal(fmt.Errorf("could not create mailer, %w", fmt.Errorf("notification must reference v1 secret containing notification configuration"))))
	})
	It("SendMail sends mail successfully", func() {
		smtpHost := "127.0.0.1"
		smtpPort := 2525
		smtpRecipients := fmt.Sprintf("%s@a.com,%s@b.com", randomString(), randomString())
		smtpSender := fmt.Sprintf("%s@d.com", randomString())
		// test server does not support auth
		emailSubject := "Test Subject"
		plainEmailMessage := "Test Message"
		htmlEmailMessage := "<html>Test Message</html>"

		smtpServer := smtpmock.New(smtpmock.ConfigurationAttr{
			HostAddress: smtpHost,
			PortNumber:  smtpPort,
		})
		if err := smtpServer.Start(); err != nil {
			Fail(fmt.Sprintf("failed to start smtp server: %v", err))
		}

		data := map[string][]byte{
			libsveltosv1beta1.SmtpRecipients: []byte(smtpRecipients),
			libsveltosv1beta1.SmtpSender:     []byte(smtpSender),
			libsveltosv1beta1.SmtpHost:       []byte(smtpHost),
			libsveltosv1beta1.SmtpPort:       []byte(fmt.Sprintf("%d", smtpPort)),
		}

		ns, sec := createNamespaceAndSecret(data)

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  ns,
				Name:       sec,
			},
		}

		mailer, err := notifications.NewMailer(context.Background(), k8sClient, notification)
		Expect(err).To(BeNil())
		Expect(mailer).ToNot(BeNil())

		err = mailer.SendMail(emailSubject, plainEmailMessage, false)
		Expect(err).To(BeNil())

		err = mailer.SendMail(emailSubject, htmlEmailMessage, true)
		Expect(err).To(BeNil())

		Eventually(func() bool {
			messages := smtpServer.MessagesAndPurge()
			if len(messages) != 2 {
				return false
			}
			if !strings.Contains(messages[0].MsgRequest(), emailSubject) {
				return false
			}
			if !strings.Contains(messages[0].MsgRequest(), plainEmailMessage) {
				return false
			}
			if !strings.Contains(messages[1].MsgRequest(), emailSubject) {
				return false
			}
			if !strings.Contains(messages[1].MsgRequest(), htmlEmailMessage) {
				return false
			}
			if !strings.Contains(messages[1].MsgRequest(), "text/html") {
				return false
			}
			return true
		}, time.Minute, time.Second).Should(BeTrue())

		if err := smtpServer.Stop(); err != nil {
			Fail(fmt.Sprintf("failed to stop smtp server: %v", err))
		}
	})
	It("getWebexInfo gets info from secret", func() {
		webexRoom := randomString()
		webexToken := randomString()

		data := map[string][]byte{
			libsveltosv1beta1.WebexRoomID: []byte(webexRoom),
			libsveltosv1beta1.WebexToken:  []byte(webexToken),
		}

		ns, sec := createNamespaceAndSecret(data)

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  ns,
				Name:       sec,
			},
		}

		webexInfo, err := notifications.GetWebexInfo(context.TODO(), k8sClient, notification)
		Expect(err).To(BeNil())
		Expect(webexInfo).ToNot(BeNil())

		room, token := notifications.ExtractWebexConfiguration(webexInfo)
		Expect(room).To(Equal(webexRoom))
		Expect(token).To(Equal(webexToken))
	})
	It("getWebexInfo raises exception if Secret Data is nil", func() {
		data := map[string][]byte{}
		ns, sec := createNamespaceAndSecret(data)

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeWebex,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  ns,
				Name:       sec,
			},
		}

		webexInfo, err := notifications.GetWebexInfo(context.TODO(), k8sClient, notification)
		Expect(webexInfo).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err).To(Equal(fmt.Errorf("notification must reference v1 secret containing notification configuration")))
	})
	It("SendWebexMessage sends message successfully", func() {
		webexRoom := fmt.Sprintf("%s", randomString())
		webexToken := fmt.Sprintf("%s", randomString())

		data := map[string][]byte{
			libsveltosv1beta1.WebexRoomID: []byte(webexRoom),
			libsveltosv1beta1.WebexToken:  []byte(webexToken),
		}

		ns, sec := createNamespaceAndSecret(data)

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeWebex,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  ns,
				Name:       sec,
			},
		}

		notifier, err := notifications.NewWebexNotifier(context.TODO(), k8sClient, notification)
		Expect(err).To(BeNil())
		Expect(notifier).ToNot(BeNil())

		notifications.MockCreateMessage(false)

		err = notifier.SendNotification("Test Message", nil, logr.New(nil))
		Expect(err).To(BeNil())
	})
	It("SendWebexMessage raises error if message was not sent successfully", func() {
		webexRoom := fmt.Sprintf("%s", randomString())
		webexToken := fmt.Sprintf("%s", randomString())

		data := map[string][]byte{
			libsveltosv1beta1.WebexRoomID: []byte(webexRoom),
			libsveltosv1beta1.WebexToken:  []byte(webexToken),
		}

		ns, sec := createNamespaceAndSecret(data)

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeWebex,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  ns,
				Name:       sec,
			},
		}

		notifier, err := notifications.NewWebexNotifier(context.TODO(), k8sClient, notification)
		Expect(err).To(BeNil())
		Expect(notifier).ToNot(BeNil())

		notifications.MockCreateMessage(true)

		err = notifier.SendNotification("Test Message", nil, logr.New(nil))
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("failed to send message"))
	})
})
