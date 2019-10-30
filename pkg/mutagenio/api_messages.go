package mutagenio

import (
	"errors"
)

// tunnelCreateRequest is the JSON format for tunnel creation requests.
type tunnelCreateRequest struct{}

// tunnelCreateResponse is the JSON format for tunnel creation responses.
type tunnelCreateResponse struct {
	// Id is the tunnel identifier.
	ID string `json:"id"`
	// HostToken is the tunnel host authorization token.
	HostToken string `json:"hostToken"`
	// ClientToken is the tunnel client authorization token.
	ClientToken string `json:"clientToken"`
}

// ensureValid ensures that tunnelCreateResponse's invariants are respected.
func (r *tunnelCreateResponse) ensureValid() error {
	// Ensure that the response is non-nil.
	if r == nil {
		return errors.New("nil response")
	}

	// Ensure that the tunnel identifier is non-empty.
	if r.ID == "" {
		return errors.New("empty tunnel identifier")
	}

	// Ensure that the host token is non-empty.
	if r.HostToken == "" {
		return errors.New("empty tunnel host token")
	}

	// Ensure that the client token is non-empty.
	if r.ClientToken == "" {
		return errors.New("empty tunnel client token")
	}

	// Success.
	return nil
}

// tunnelHostExchangeRequest is the JSON format for tunnel host exchange
// requests.
type tunnelHostExchangeRequest struct {
	// Offer is the tunnel host's offer.
	Offer string `json:"offer"`
	// Signature is the signature for the tunnel host's offer.
	Signature string `json:"signature"`
}

// tunnelHostExchangeResponse is the JSON format for tunnel host exchange
// responses.
type tunnelHostExchangeResponse struct {
	// Offer is the tunnel client's offer.
	Offer string `json:"offer"`
	// Signature is the signature for the tunnel client's offer.
	Signature string `json:"signature"`
}

// ensureValid ensures that tunnelHostExchangeResponse's invariants are
// respected.
func (r *tunnelHostExchangeResponse) ensureValid() error {
	// Ensure that the response is non-nil.
	if r == nil {
		return errors.New("nil response")
	}

	// Ensure that the tunnel client's offer is non-empty.
	if r.Offer == "" {
		return errors.New("empty tunnel client offer")
	}

	// Ensure that the tunnel client's offer signature is non-empty.
	if r.Signature == "" {
		return errors.New("empty tunnel client offer signature")
	}

	// Success.
	return nil
}

// tunnelClientExchangeStartRequest is the JSON format for tunnel client
// exchange initiation requests.
type tunnelClientExchangeStartRequest struct{}

// tunnelClientExchangeStartResponse is the JSON format for tunnel client
// exchange initiation responses.
type tunnelClientExchangeStartResponse struct {
	// ExchangeID is the identifier for the exchange operation.
	ExchangeID string `json: "exchangeId"`
	// Offer is the tunnel host's offer.
	Offer string `json:"offer"`
	// Signature is the signature for the tunnel host's offer.
	Signature string `json:"signature"`
}

// ensureValid ensures that tunnelClientExchangeStartResponse's invariants are
// respected.
func (r *tunnelClientExchangeStartResponse) ensureValid() error {
	// Ensure that the response is non-nil.
	if r == nil {
		return errors.New("nil response")
	}

	// Ensure that the exchange identifier is non-empty.
	if r.ExchangeID == "" {
		return errors.New("empty exchange identifier")
	}

	// Ensure that the tunnel host's offer is non-empty.
	if r.Offer == "" {
		return errors.New("empty tunnel host offer")
	}

	// Ensure that the tunnel host's offer signature is non-empty.
	if r.Signature == "" {
		return errors.New("empty tunnel host offer signature")
	}

	// Success.
	return nil
}

// tunnelClientExchangeFinishRequest is the JSON format for tunnel client
// exchange completion requests.
type tunnelClientExchangeFinishRequest struct {
	// ExchangeID is the identifier for the exchange operation.
	ExchangeID string `json: "exchangeId"`
	// Offer is the tunnel client's offer.
	Offer string `json:"offer"`
	// Signature is the signature for the tunnel client's offer.
	Signature string `json:"signature"`
}

// tunnelClientExchangeFinishResponse is the JSON format for tunnel client
// exchange completion responses.
type tunnelClientExchangeFinishResponse struct{}

// ensureValid ensures that tunnelClientExchangeFinishResponse's invariants are
// respected.
func (r *tunnelClientExchangeFinishResponse) ensureValid() error {
	// Ensure that the response is non-nil.
	if r == nil {
		return errors.New("nil response")
	}

	// Success.
	return nil
}
