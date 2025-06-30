package entities

type StripeSessionResponse struct {
	Code      string `json:"code"`
	URL       string `json:"url"`
	SessionID string `json:"session_id"`
}
