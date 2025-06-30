package entities

type StripeSessionResponse struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}
