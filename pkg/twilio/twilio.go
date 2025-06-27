package twilio

import (
	"errors"

	"github.com/Resolyte-official/service-governance/pkg/config"
	svcError "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/twilio/twilio-go"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
)

type TwilioClient interface {
	SendOTP(phone string) error
	VerifyOTP(phone, otp string) error
}

type twilioClient struct {
	client           *twilio.RestClient
	verifyServiceSid string
}

func NewTwilioClient(envConfig *config.Configuration) TwilioClient {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username:   envConfig.TwilioApiKey,
		Password:   envConfig.TwilioApiSecret,
		AccountSid: envConfig.TwilioAccountSid,
	})
	client.SetRegion("gll")
	client.SetEdge("roaming")
	return &twilioClient{
		client:           client,
		verifyServiceSid: envConfig.TwilioVerifyServiceSid,
	}
}

func (tc *twilioClient) SendOTP(phone string) error {
	params := &verify.CreateVerificationParams{}
	params.SetTo(phone)
	params.SetChannel("sms")

	_, err := tc.client.VerifyV2.CreateVerification(tc.verifyServiceSid, params)
	if err != nil {
		return err
	}
	return nil
}

func (tc *twilioClient) VerifyOTP(phone, otp string) error {
	params := &verify.CreateVerificationCheckParams{}
	params.SetTo(phone)
	params.SetCode(otp)

	resp, err := tc.client.VerifyV2.CreateVerificationCheck(tc.verifyServiceSid, params)
	if err != nil {
		return err
	}
	if resp.Status != nil && *(resp.Status) != "approved" {
		return svcError.ErrUnauthorized.WithInternalErr(errors.New("verification failed"))
	}
	return nil
}
