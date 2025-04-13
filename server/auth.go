package server

import (
	"context"
	"github.com/auth0/go-auth0/authentication"
	"github.com/auth0/go-auth0/management"
	"github.com/spf13/viper"
)

type AuthenticationManager struct {
	// auth - Used for authenticating users with the API
	auth *authentication.Authentication

	// management - Use for managing users and roles in Auth0
	management *management.Management
}

/*
NewAuthenticationManager - Instantiates a new AuthenticationManager structure
*/
func NewAuthenticationManager(domain string, clientId string, clientSecret string) (*AuthenticationManager, error) {
	authAPI, err := authentication.New(
		context.Background(),
		domain,
		authentication.WithClientID(clientId),
		authentication.WithClientSecret(clientSecret),
	)
	if err != nil {
		return nil, err
	}

	managementAPI, err := management.New(
		domain,
		management.WithClientCredentials(context.Background(), clientId, clientSecret),
	)
	if err != nil {
		return nil, err
	}

	return &AuthenticationManager{auth: authAPI, management: managementAPI}, nil
}

/*
NewAuthenticationManagerFromConfig - Instantiates a new AuthenticationManager using values provided in viper
*/
func NewAuthenticationManagerFromConfig() (*AuthenticationManager, error) {
	auth, err := NewAuthenticationManager(
		viper.GetString("auth0.domain"),
		viper.GetString("auth0.client_id"),
		viper.GetString("auth0.client_secret"))
	if err != nil {
		return nil, err
	}

	return auth, nil
}
