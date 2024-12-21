package user

import (
	"regexp"

	"github.com/auth0/go-auth0/authentication/database"
	"github.com/auth0/go-auth0/authentication/oauth"
	"github.com/spf13/viper"
	"github.com/stevezaluk/mtgjson-models/errors"
	"github.com/stevezaluk/mtgjson-models/user"
	mtgContext "github.com/stevezaluk/mtgjson-sdk/context"
	"go.mongodb.org/mongo-driver/bson"

	"context"
)

const SCOPE = "read:user write:user read:card write:card read:set write:set read:deck write:deck read:health read:metrics"

/*
Ensures that the passed string is a valid email address. If the email address is not valid then it returns false,
true otherwise
*/
func validateEmail(email string) bool {
	var result bool

	reg, _ := regexp.Compile(`^[\w\.-]+@[a-zA-Z\d\.-]+\.[a-zA-Z]{2,}$`)
	if reg.MatchString(email) {
		result = true
	}

	return result
}

/*
Fetch a user based on there username. Returns ErrNoUser if the user cannot be found
*/
func GetUser(email string) (*user.User, error) {
	var result *user.User

	if email == "" {
		return result, errors.ErrUserMissingId
	}

	if !validateEmail(email) {
		return result, errors.ErrInvalidEmail
	}

	var mongoDatabase = mtgContext.GetDatabase()

	query := bson.M{"email": email}
	results := mongoDatabase.Find("user", query, &result)
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
	if user.Username == "" || user.Email == "" || user.Auth0Id == "" {
		return errors.ErrUserMissingId
	}

	if !validateEmail(user.Email) {
		return errors.ErrInvalidEmail
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
List all users from the database, and return them in a slice. A limit can be provided to ensure that too many objects
dont get returned
*/
func IndexUsers(limit int64) ([]user.User, error) {
	var result []user.User

	var database = mtgContext.GetDatabase()

	results := database.Index("user", limit, &result)
	if results == nil {
		return result, errors.ErrNoUser
	}

	return result, nil
}

/*
Removes the requested users account from the MongoDB database. Does not remove there account from Auth0. Returns ErrUserMissingId if email is empty string,
returns ErrInvalidEmail if the email address passed is not valid, returns ErrUserDeleteFailed if the DeletedCount is less than 1, and returns nil otherwise
*/
func DeleteUser(email string) error {
	_, err := GetUser(email)
	if err != nil {
		return err
	}

	var database = mtgContext.GetDatabase()

	results := database.Delete("user", bson.M{"email": email})
	if results.DeletedCount > 1 {
		return errors.ErrUserDeleteFailed
	}

	return nil
}

/*
Register a new user with Auth0 and store there user model within the MongoDB database
*/
func RegisterUser(username string, email string, password string) (user.User, error) {
	var ret user.User

	ret.Username = username
	ret.Email = email

	if !validateEmail(email) {
		return ret, errors.ErrInvalidEmail
	}

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

	userResp, err := authAPI.Database.Signup(context.Background(), userData)
	if err != nil {
		return ret, errors.ErrFailedToRegisterUser
	}

	ret.Auth0Id = userResp.ID

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
		Audience: viper.GetString("auth0.audience"),
		Scope:    SCOPE,
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

/*
Completely removes the requested user account, both from Auth0 and from MongoDB
*/
func DeactivateUser(email string) error {
	user, err := GetUser(email)
	if err != nil {
		return err
	}

	err = DeleteUser(email)
	if err != nil {
		return err
	}

	var managementAPI = mtgContext.GetAuthManagementAPI()

	userId := "auth0|" + user.Auth0Id

	err = managementAPI.User.Delete(context.TODO(), userId)
	if err != nil {
		return err
	}

	return nil
}

/*
Send a reset password email to a specified user acocunt.
*/
func ResetUserPassword(email string) error {
	_, err := GetUser(email)
	if err != nil {
		return err
	}

	var authAPI = mtgContext.GetAuthAPI()

	resetPwdRequest := database.ChangePasswordRequest{
		Email:      email,
		Connection: "Username-Password-Authentication",
	}

	_, err = authAPI.Database.ChangePassword(
		context.Background(),
		resetPwdRequest,
	)

	if err != nil {
		return err
	}

	return nil
}
