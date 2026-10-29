package akeylessapi

import (
	"context"
	"fmt"
	"strings"

	akeyless "github.com/akeylesslabs/akeyless-go/v3"
)

// Authenticate calls Akeyless /auth with the given body and returns the API token.
func Authenticate(ctx context.Context, apiURL string, body akeyless.Auth) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	client := akeyless.NewAPIClient(&akeyless.Configuration{
		Servers: akeyless.ServerConfigurations{{URL: strings.TrimSuffix(apiURL, "/")}},
	})

	out, _, err := client.V2Api.Auth(ctx).Body(body).Execute()
	if err != nil {
		return "", fmt.Errorf("akeyless auth: %w", err)
	}
	token := out.GetToken()
	if token == "" {
		return "", fmt.Errorf("akeyless auth: empty token in response")
	}
	return token, nil
}
