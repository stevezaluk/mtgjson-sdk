package user

import (
	"github.com/auth0/go-auth0/authentication/database"
	"github.com/auth0/go-auth0/authentication/oauth"
	"github.com/stevezaluk/mtgjson-models/errors"
	"github.com/stevezaluk/mtgjson-models/user"
	mtgContext "github.com/stevezaluk/mtgjson-sdk/context"
	"go.mongodb.org/mongo-driver/bson"

	"context"
)

/*
Fetch a user based on there username. Returns ErrNoUser if the user cannot be found
*/
func GetUser(email string) (user.User, error) {
	var result user.User

	var database = mtgContext.GetDatabase()

	query := bson.M{"email": email}
	results := database.Find("user", query, &result)
	if results == nil {
		return result, errors.ErrNoUser
	}

	return result, nil
}

/*
Insert the contents of a User model in the MongoDB database. Returns ErrUserMissingId if the Username, or Emai is not present
Returns ErrUserAlreadyExist if a user already exists under this username
*/
func NewUser(user user.User) error {
	if user.Username == "" || user.Email == "" {
		return errors.ErrUserMissingId
	}

	_, err := GetUser(user.Email)
	if err != errors.ErrNoUser {
		return errors.ErrUserAlreadyExist
	}

	var database = mtgContext.GetDatabase()
	database.Insert("user", &user)

	return nil
}

/*
Register a new user with Auth0 and store there user model within the MongoDB database
*/
func RegisterUser(username string, email string, password string) (user.User, error) {
	var ret user.User

	ret.Username = username
	ret.Email = email

	if len(password) < 12 {
		return ret, errors.ErrInvalidPasswordLength
	}

	userData := database.SignupRequest{
		Connection: "Username-Password-Authentication",
		Username:   ret.Username,
		Password:   password,
		Email:      ret.Email,
	}

	authAPI := mtgContext.GetAuthAPI()

	_, err := authAPI.Database.Signup(context.Background(), userData)
	if err != nil {
		return ret, errors.ErrFailedToRegisterUser
	}

	err = NewUser(ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

/*
Log a user in with there email address and password and return back a oauth.TokenSet
*/
func LoginUser(email string, password string) (*oauth.TokenSet, error) {
	_, err := GetUser(email)
	if err != nil {
		return nil, err
	}

	authAPI := mtgContext.GetAuthAPI()

	userData := oauth.LoginWithPasswordRequest{
		Username: email,
		Password: password,
	}

	validateOpts := oauth.IDTokenValidationOptions{}

	token, err := authAPI.OAuth.LoginWithPassword(
		context.Background(),
		userData,
		validateOpts,
	)

	if err != nil {
		return token, err
	}

	return token, nil
}
