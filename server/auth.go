package server

import (
	"context"
	"github.com/auth0/go-auth0/authentication"
	"github.com/auth0/go-auth0/authentication/database"
	"github.com/auth0/go-auth0/authentication/oauth"
	"github.com/auth0/go-auth0/management"
	"github.com/spf13/viper"
)

type AuthenticationManager struct {
	// audience - The audience of your Auth0 API
	audience string

	// domain - The Auth0 domain of your tenant
	domain string

	// scope - A string of space seperated scopes that the API should recognize
	scope string

	// auth - Used for authenticating users with the API
	auth *authentication.Authentication

	// management - Use for managing users and roles in Auth0
	management *management.Management
}

/*
NewAuthenticationManager - Instantiates a new AuthenticationManager structure
*/
func NewAuthenticationManager(domain string, audience string, clientId string, clientSecret string) (*AuthenticationManager, error) {
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

	return &AuthenticationManager{
		audience:   audience,
		domain:     domain,
		auth:       authAPI,
		management: managementAPI}, nil
}

/*
NewAuthenticationManagerFromConfig - Instantiates a new AuthenticationManager using values provided in viper
*/
func NewAuthenticationManagerFromConfig() (*AuthenticationManager, error) {
	auth, err := NewAuthenticationManager(
		viper.GetString("auth0.domain"),
		viper.GetString("auth0.audience"),
		viper.GetString("auth0.client_id"),
		viper.GetString("auth0.client_secret"))
	if err != nil {
		return nil, err
	}

	auth.SetScope(viper.GetString("auth0.scope"))

	return auth, nil
}

/*
SetScope - Set the scopes that the API should recognize
*/
func (auth *AuthenticationManager) SetScope(scope string) {
	auth.scope = scope
}

/*
GetEmailFromToken - Calls the /userinfo OIDC endpoint and return the users email address
*/
func (auth *AuthenticationManager) GetEmailFromToken(token string) (string, error) {
	userInfo, err := auth.auth.UserInfo(context.Background(), token)
	if err != nil {
		return "", err
	}

	return userInfo.Email, nil

}

/*
AuthenticateUser - Fetch a token on the users behalf using there credentials
*/
func (auth *AuthenticationManager) AuthenticateUser(username string, password string) (*oauth.TokenSet, error) {
	request := oauth.LoginWithPasswordRequest{
		Username: username,
		Password: password,
		Audience: auth.audience,
		Scope:    auth.scope,
	}

	token, err := auth.auth.OAuth.LoginWithPassword(
		context.Background(),
		request,
		oauth.IDTokenValidationOptions{})
	if err != nil {
		return nil, err
	}

	return token, nil
}

/*
RegisterUser - Register a user with Auth0. This does not create a mtgjson-api user, only a user within MTGJSON.
You should configure an Auth0 auth flow for automatically assigning users a role
*/
func (auth *AuthenticationManager) RegisterUser(username string, email string, password string) (*database.SignupResponse, error) {
	request := database.SignupRequest{
		Connection: "Username-Password-Authentication",
		Username:   username,
		Email:      email,
		Password:   password,
	}

	userResp, err := auth.auth.Database.Signup(context.Background(), request)
	if err != nil {
		return nil, err
	}

	return userResp, err
}

/*
ResetUserPassword - Trigger a reset password email to be sent to the users account
*/
func (auth *AuthenticationManager) ResetUserPassword(email string) error {
	request := database.ChangePasswordRequest{
		Email:      email,
		Connection: "Username-Password-Authentication",
	}

	_, err := auth.auth.Database.ChangePassword(context.Background(), request)
	if err != nil {
		return err
	}

	return nil
}

/*
DeactivateUser - Delete a user from Auth0. This does not remove the user from mtgjson-api, only Auth0
*/
func (auth *AuthenticationManager) DeactivateUser(email string) error {
	err := auth.management.User.Delete(context.Background(), email)
	if err != nil {
		return err
	}

	return nil
}
