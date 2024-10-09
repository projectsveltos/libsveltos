package notifications_test

import (
	"context"
	"fmt"

	smtpmock "github.com/mocktools/go-smtp-mock/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		secretNamespace := randomString()
		secretNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretNamespace,
			},
		}
		Expect(k8sClient.Create(context.TODO(), secretNs)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, secretNs)).To(Succeed())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: secretNamespace,
			},
			Data: map[string][]byte{
				libsveltosv1beta1.SmtpRecipients: []byte(smtpRecipients),
				libsveltosv1beta1.SmtpBcc:        []byte(smtpBcc),
				libsveltosv1beta1.SmtpIdentity:   []byte(smtpIdentity),
				libsveltosv1beta1.SmtpSender:     []byte(smtpSender),
				libsveltosv1beta1.SmtpPassword:   []byte(smtpPassword),
				libsveltosv1beta1.SmtpHost:       []byte(smtpHost),
				libsveltosv1beta1.SmtpPort:       []byte(fmt.Sprintf("%d", smtpPort)),
			},
		}

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  secret.Namespace,
				Name:       secret.Name,
			},
		}

		Expect(k8sClient.Create(context.TODO(), secret)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, secret)).To(Succeed())

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
		secretNamespace := randomString()
		secretNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretNamespace,
			},
		}
		Expect(k8sClient.Create(context.TODO(), secretNs)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, secretNs)).To(Succeed())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: secretNamespace,
			},
		}

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  secret.Namespace,
				Name:       secret.Name,
			},
		}

		Expect(k8sClient.Create(context.TODO(), secret)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, secret)).To(Succeed())

		smptInfo, err := notifications.GetSmtpInfo(context.TODO(), k8sClient, notification)
		Expect(smptInfo).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err).To(Equal(fmt.Errorf("notification must reference v1 secret containing smtp configuration")))
	})
	It("getSmtpInfo raises exception if NotificationRef is nil", func() {
		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
		}

		_, err := notifications.GetSmtpInfo(context.TODO(), k8sClient, notification)
		Expect(err).ToNot(BeNil())
		Expect(err).To(Equal(fmt.Errorf("notification must reference v1 secret containing smtp configuration")))
	})
	It("NewMailer creates a new mailer", func() {
		secretNamespace := randomString()
		secretNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretNamespace,
			},
		}
		Expect(k8sClient.Create(context.TODO(), secretNs)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, secretNs)).To(Succeed())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: secretNamespace,
			},
			Data: map[string][]byte{
				libsveltosv1beta1.SmtpRecipients: []byte(fmt.Sprintf("%s@a.com,%s@b.com", randomString(), randomString())),
				libsveltosv1beta1.SmtpBcc:        []byte(fmt.Sprintf("%s@c.com", randomString())),
				libsveltosv1beta1.SmtpIdentity:   []byte(randomString()),
				libsveltosv1beta1.SmtpSender:     []byte(fmt.Sprintf("%s@d.com", randomString())),
				libsveltosv1beta1.SmtpPassword:   []byte(randomString()),
				libsveltosv1beta1.SmtpHost:       []byte(fmt.Sprintf("%s.com", randomString())),
				libsveltosv1beta1.SmtpPort:       []byte(fmt.Sprintf("%d", rand.IntnRange(444, 9999))),
			},
		}

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  secret.Namespace,
				Name:       secret.Name,
			},
		}

		Expect(k8sClient.Create(context.TODO(), secret)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, secret)).To(Succeed())

		mailer, err := notifications.NewMailer(context.Background(), k8sClient, notification)
		Expect(err).To(BeNil())
		Expect(mailer).ToNot(BeNil())
	})
	It("NewMailer raises exception if notification is nil", func() {
		secretNamespace := randomString()
		secretNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretNamespace,
			},
		}
		Expect(k8sClient.Create(context.TODO(), secretNs)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, secretNs)).To(Succeed())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: secretNamespace,
			},
			Data: map[string][]byte{
				libsveltosv1beta1.SmtpRecipients: []byte(fmt.Sprintf("%s@a.com,%s@b.com", randomString(), randomString())),
				libsveltosv1beta1.SmtpBcc:        []byte(fmt.Sprintf("%s@c.com", randomString())),
				libsveltosv1beta1.SmtpIdentity:   []byte(randomString()),
				libsveltosv1beta1.SmtpSender:     []byte(fmt.Sprintf("%s@d.com", randomString())),
				libsveltosv1beta1.SmtpPassword:   []byte(randomString()),
				libsveltosv1beta1.SmtpHost:       []byte(fmt.Sprintf("%s.com", randomString())),
				libsveltosv1beta1.SmtpPort:       []byte(fmt.Sprintf("%d", rand.IntnRange(444, 9999))),
			},
		}

		Expect(k8sClient.Create(context.TODO(), secret)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, secret)).To(Succeed())

		mailer, err := notifications.NewMailer(context.Background(), k8sClient, &libsveltosv1beta1.Notification{})
		Expect(mailer).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err).To(Equal(fmt.Errorf("could not create mailer, %w", fmt.Errorf("notification must reference v1 secret containing smtp configuration"))))
	})
	It("SendMail sends mail successfully", func() {
		smtpHost := "127.0.0.1"
		smtpPort := 2525
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

		secretNamespace := randomString()
		secretNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: secretNamespace,
			},
		}
		Expect(k8sClient.Create(context.TODO(), secretNs)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, secretNs)).To(Succeed())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      randomString(),
				Namespace: secretNamespace,
			},
			Data: map[string][]byte{
				libsveltosv1beta1.SmtpRecipients: []byte(fmt.Sprintf("%s@a.com,%s@b.com", randomString(), randomString())),
				libsveltosv1beta1.SmtpSender:     []byte(fmt.Sprintf("%s@d.com", randomString())),
				libsveltosv1beta1.SmtpHost:       []byte(smtpHost),
				libsveltosv1beta1.SmtpPort:       []byte(fmt.Sprintf("%d", smtpPort)),
			},
		}

		notification := &libsveltosv1beta1.Notification{
			Name: randomString(),
			Type: libsveltosv1beta1.NotificationTypeSMTP,
			NotificationRef: &corev1.ObjectReference{
				Kind:       "Secret",
				APIVersion: "v1",
				Namespace:  secret.Namespace,
				Name:       secret.Name,
			},
		}

		Expect(k8sClient.Create(context.TODO(), secret)).To(Succeed())
		Expect(waitForObject(context.TODO(), k8sClient, secret)).To(Succeed())

		mailer, err := notifications.NewMailer(context.Background(), k8sClient, notification)
		Expect(err).To(BeNil())
		Expect(mailer).ToNot(BeNil())

		err = mailer.SendMail(emailSubject, plainEmailMessage, false)
		Expect(err).To(BeNil())

		err = mailer.SendMail(emailSubject, htmlEmailMessage, true)
		Expect(err).To(BeNil())

		messages := smtpServer.MessagesAndPurge()
		Expect(len(messages)).To(Equal(2))
		Expect(messages[0].MsgRequest()).To(ContainSubstring(emailSubject))
		Expect(messages[0].MsgRequest()).To(ContainSubstring(plainEmailMessage))
		Expect(messages[1].MsgRequest()).To(ContainSubstring(emailSubject))
		Expect(messages[1].MsgRequest()).To(ContainSubstring(htmlEmailMessage))
		Expect(messages[1].MsgRequest()).To(ContainSubstring("text/html"))

		if err := smtpServer.Stop(); err != nil {
			Fail(fmt.Sprintf("failed to stop smtp server: %v", err))
		}
	})
})
