package main

import (
	"context"

	"github.com/compozy/agh/internal/bridgesdk"
)

func sendWhatsAppDeliveryMessage(
	ctx context.Context,
	api whatsappAPI,
	phoneNumberID string,
	request whatsappSendMessageRequest,
) (*whatsappSendMessageResponse, error) {
	return bridgesdk.RetryDo(
		ctx,
		bridgesdk.DefaultRetryConfig(),
		func(callCtx context.Context) (*whatsappSendMessageResponse, error) {
			return api.SendTextMessage(callCtx, phoneNumberID, request)
		},
	)
}
