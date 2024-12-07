package notifications

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"

	"github.com/go-logr/logr"
	webexteams "github.com/jbogarin/go-cisco-webex-teams/sdk"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

type webexInfo struct {
	token string
	room  string
}

type WebexNotifier struct {
	webexInfo *webexInfo
}

var (
	sendWebexMessage = func(wc *webexteams.Client, message *webexteams.MessageCreateRequest) (*webexteams.Message, *resty.Response, error) {
		return wc.Messages.CreateMessage(message)
	}
)

func NewWebexNotifier(ctx context.Context, c client.Client, notification *libsveltosv1beta1.Notification) (*WebexNotifier, error) {
	info, err := getWebexInfo(ctx, c, notification)
	if err != nil {
		return nil, err
	}
	return &WebexNotifier{webexInfo: info}, nil
}

func (wn *WebexNotifier) SendNotification(message string, files []webexteams.File, logger logr.Logger) error {
	l := logger.WithValues("room", wn.webexInfo.room)
	l.V(logs.LogInfo).Info("send webex message")

	webexClient := webexteams.NewClient()
	if webexClient == nil {
		l.V(logs.LogInfo).Info("failed to get webexClient client")
		return fmt.Errorf("failed to get webexClient client")
	}
	webexClient.SetAuthToken(wn.webexInfo.token)

	webexMessage := &webexteams.MessageCreateRequest{
		Markdown: message,
		RoomID:   wn.webexInfo.room,
		Files:    files,
	}

	_, resp, err := sendWebexMessage(webexClient, webexMessage)

	if err != nil {
		l.V(logs.LogInfo).Info(fmt.Sprintf("Failed to send message. Error: %v", err))
		return err
	}

	if resp != nil {
		l.V(logs.LogDebug).Info(fmt.Sprintf("response: %s", string(resp.Body())))
	}

	return nil
}

func getWebexInfo(ctx context.Context, c client.Client, notification *libsveltosv1beta1.Notification) (*webexInfo, error) {
	secret, err := getSecret(ctx, c, notification)
	if err != nil {
		return nil, err
	}

	authToken, ok := secret.Data[libsveltosv1beta1.WebexToken]
	if !ok {
		return nil, fmt.Errorf("secret does not contain webex token")
	}

	room, ok := secret.Data[libsveltosv1beta1.WebexRoomID]
	if !ok {
		return nil, fmt.Errorf("secret does not contain webex room")
	}

	return &webexInfo{token: string(authToken), room: string(room)}, nil
}
