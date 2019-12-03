package mutagenio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

const (
	// apiEndpoint is the mutagen.io API endpoint.
	apiEndpoint = "https://api.mutagen.io/v1/"
	// apiEndpoint is the development API endpoint.
	apiEndpointDevelopment = "http://localhost:8081/v1/"
)

// unexpectedResponseStatusError is an error type returned from callAPI if the
// response status returned from the server did not match what was expected.
type unexpectedResponseStatusError struct {
	// status is the returned status.
	status int
}

// Error implements error.Error.
func (e *unexpectedResponseStatusError) Error() string {
	return fmt.Sprintf("unexpected response status: %d %s", e.status, http.StatusText(e.status))
}

var (
	// ErrUnauthorized is a sentinel error returned if an API endpoint returns
	// a 401 (Unauthorized) status code.
	ErrUnauthorized = errors.New("unauthorized")
)

// callAPI performs an API request using the specified parameters.
func callAPI(ctx context.Context, method, resource, token string, request, response interface{}, expectedStatus int) error {
	// Create a buffer for the body and encode the request object into it.
	body := &bytes.Buffer{}
	encoder := json.NewEncoder(body)
	if err := encoder.Encode(request); err != nil {
		return fmt.Errorf("unable to encode request body: %w", err)
	}

	// Compute the API endpoint.
	apiEndpoint := apiEndpoint
	if mutagen.DevelopmentModeEnabled {
		apiEndpoint = apiEndpointDevelopment
	}

	// Create the request.
	httpRequest, err := http.NewRequestWithContext(ctx, method, apiEndpoint+resource, body)
	if err != nil {
		return fmt.Errorf("unable to create request: %w", err)
	}

	// Set the authorization header.
	httpRequest.Header.Set("Authorization", "Bearer "+token)

	// Perform the request and (if successful) ensure the response body is
	// closed.
	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("unable to perform request: %w", err)
	}
	defer httpResponse.Body.Close()

	// Verify that the response status code is as expected.
	if httpResponse.StatusCode != expectedStatus {
		return &unexpectedResponseStatusError{httpResponse.StatusCode}
	}

	// Decode the response body. We allow unknown fields to ensure that we're
	// forward-compatible with extended API responses.
	decoder := json.NewDecoder(httpResponse.Body)
	if err := decoder.Decode(response); err != nil {
		return fmt.Errorf("unable to decode response body: %w", err)
	}

	// Success.
	return nil
}

// TunnelCreate creates a new tunnel using the underlying API key. It returns
// the tunnel identifier, host token, and client token.
func TunnelCreate(ctx context.Context) (string, string, string, error) {
	// Read the API token.
	apiToken, err := readAPIToken()
	if err != nil {
		return "", "", "", fmt.Errorf("unable to read API token: %w", err)
	}

	// Create the request.
	request := &tunnelCreateRequest{}

	// Perform the operation and validate the response.
	response := &tunnelCreateResponse{}
	if err := callAPI(ctx, http.MethodPost, "tunnels", apiToken, request, response, http.StatusCreated); err != nil {
		if unexpectedResponseStatusErr, ok := err.(*unexpectedResponseStatusError); ok {
			if unexpectedResponseStatusErr.status == http.StatusUnauthorized {
				return "", "", "", ErrUnauthorized
			}
		}
		return "", "", "", err
	} else if err = response.ensureValid(); err != nil {
		return "", "", "", fmt.Errorf("received invalid response: %w", err)
	}

	// Success.
	return response.ID, response.HostToken, response.ClientToken, nil
}

// TunnelHostExchange performs the host side of a tunnel offer exchange.
func TunnelHostExchange(ctx context.Context, tunnelID, hostToken, offer, signature string) (string, string, error) {
	// Create the request.
	request := &tunnelHostExchangeRequest{
		Offer:     offer,
		Signature: signature,
	}

	// Perform the operation and validate the response.
	response := &tunnelHostExchangeResponse{}
	if err := callAPI(ctx, http.MethodPost, "tunnels/"+tunnelID, hostToken, request, response, http.StatusOK); err != nil {
		if unexpectedResponseStatusErr, ok := err.(*unexpectedResponseStatusError); ok {
			if unexpectedResponseStatusErr.status == http.StatusUnauthorized {
				return "", "", ErrUnauthorized
			}
		}
		return "", "", err
	} else if err = response.ensureValid(); err != nil {
		return "", "", fmt.Errorf("received invalid response: %w", err)
	}

	// Success.
	return response.Offer, response.Signature, nil
}

// TunnelClientExchangeStart performs initiation of the client side of a tunnel
// offer exchange.
func TunnelClientExchangeStart(ctx context.Context, tunnelID, clientToken string) (string, string, string, error) {
	// Create the request.
	request := &tunnelClientExchangeStartRequest{}

	// Perform the operation and validate the response.
	response := &tunnelClientExchangeStartResponse{}
	if err := callAPI(ctx, http.MethodGet, "tunnels/"+tunnelID, clientToken, request, response, http.StatusOK); err != nil {
		if unexpectedResponseStatusErr, ok := err.(*unexpectedResponseStatusError); ok {
			if unexpectedResponseStatusErr.status == http.StatusUnauthorized {
				return "", "", "", ErrUnauthorized
			}
		}
		return "", "", "", err
	} else if err = response.ensureValid(); err != nil {
		return "", "", "", fmt.Errorf("received invalid response: %w", err)
	}

	// Success.
	return response.ExchangeID, response.Offer, response.Signature, nil
}

// TunnelClientExchangeFinish performs completion of the client side of a tunnel
// offer exchange.
func TunnelClientExchangeFinish(ctx context.Context, tunnelID, clientToken, exchangeID, offer, signature string) error {
	// Create the request.
	request := &tunnelClientExchangeFinishRequest{
		ExchangeID: exchangeID,
		Offer:      offer,
		Signature:  signature,
	}

	// Perform the operation and validate the response.
	response := &tunnelClientExchangeFinishResponse{}
	if err := callAPI(ctx, http.MethodPut, "tunnels/"+tunnelID, clientToken, request, response, http.StatusOK); err != nil {
		if unexpectedResponseStatusErr, ok := err.(*unexpectedResponseStatusError); ok {
			if unexpectedResponseStatusErr.status == http.StatusUnauthorized {
				return ErrUnauthorized
			}
		}
		return err
	} else if err = response.ensureValid(); err != nil {
		return fmt.Errorf("received invalid response: %w", err)
	}

	// Success.
	return nil
}
