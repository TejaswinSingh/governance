package twilio

import (
	"errors"

	"github.com/Resolyte-official/service-governance/pkg/config"
	svcError "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/twilio/twilio-go"
	"github.com/twilio/twilio-go/client"
	lookups "github.com/twilio/twilio-go/rest/lookups/v1"
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
		Username: envConfig.TwilioAccountSid,
		Password: envConfig.TwilioAuthToken,
	})
	return &twilioClient{
		client:           client,
		verifyServiceSid: envConfig.TwilioVerifyServiceSid,
	}
}

func (tc *twilioClient) IsPhoneValid(phone string) bool {
	params := &lookups.FetchPhoneNumberParams{}
	// params.SetType([]string{"carrier"}) // Optional, gets carrier/type info

	if _, err := tc.client.LookupsV1.FetchPhoneNumber(phone, params); err != nil {
		return false
	}

	return true
}

func (tc *twilioClient) SendOTP(phone string) error {
	params := &verify.CreateVerificationParams{}
	params.SetTo(phone)
	params.SetChannel("sms")

	_, err := tc.client.VerifyV2.CreateVerification(tc.verifyServiceSid, params)
	if err != nil {
		return svcError.ErrServiceUnavailable.WithInternalErr(err)
	}
	return nil
}

func (tc *twilioClient) VerifyOTP(phone, otp string) error {
	params := &verify.CreateVerificationCheckParams{}
	params.SetTo(phone)
	params.SetCode(otp)

	resp, err := tc.client.VerifyV2.CreateVerificationCheck(tc.verifyServiceSid, params)
	if err != nil {
		var restErr *client.TwilioRestError
		if errors.As(err, &restErr) {
			if restErr.Code == 20404 {
				return svcError.ErrInvalidOTP
			}
		}
		return svcError.ErrServiceUnavailable.WithInternalErr(err)
	}
	if resp != nil && resp.Status != nil && *(resp.Status) == "approved" {
		return nil
	}
	return svcError.ErrServiceUnavailable.WithInternalErr(errors.New("twilio REST API error"))
}
