package user

import (
	"errors"
	"os/user"
	"regexp"

	"github.com/auth0/go-auth0/authentication/database"
	"github.com/auth0/go-auth0/authentication/oauth"
	"github.com/spf13/viper"
	sdkErrors "github.com/stevezaluk/mtgjson-models/errors"
	userModel "github.com/stevezaluk/mtgjson-models/user"
	mtgContext "github.com/stevezaluk/mtgjson-sdk/context"
	"go.mongodb.org/mongo-driver/bson"

	"context"
)

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
GetUser Fetch a user based on their username. Returns ErrNoUser if the user cannot be found
*/
func GetUser(email string) (*userModel.User, error) {
	var result *userModel.User

	if email == "" {
		return nil, sdkErrors.ErrUserMissingId
	}

	if !validateEmail(email) {
		return nil, sdkErrors.ErrInvalidEmail
	}

	var mongoDatabase = mtgContext.GetDatabase()

	query := bson.M{"email": email}
	err := mongoDatabase.Find("user", query, &result)
	if !err {
		return nil, sdkErrors.ErrNoUser
	}

	return result, nil
}

/*
NewUser Insert the contents of a User model in the MongoDB database. Returns ErrUserMissingId if the Username, or Email is not present
Returns ErrUserAlreadyExist if a user already exists under this username
*/
func NewUser(user *userModel.User) error {
	if user.Username == "" || user.Email == "" || user.Auth0Id == "" {
		return sdkErrors.ErrUserMissingId
	}

	if !validateEmail(user.Email) {
		return sdkErrors.ErrInvalidEmail
	}

	_, err := GetUser(user.Email)
	if !errors.Is(err, sdkErrors.ErrNoUser) {
		return sdkErrors.ErrUserAlreadyExist
	}

	var mongoDatabase = mtgContext.GetDatabase()
	mongoDatabase.Insert("user", &user)

	return nil
}

/*
IndexUsers List all users from the database, and return them in a slice. A limit can be provided to ensure that too many objects
don't get returned
*/
func IndexUsers(limit int64) ([]*user.User, error) {
	var result []*user.User

	var mongoDatabase = mtgContext.GetDatabase()

	err := mongoDatabase.Index("user", limit, &result)
	if !err {
		return nil, sdkErrors.ErrNoUser
	}

	return result, nil
}

/*
DeleteUser Removes the requested users account from the MongoDB database. Does not remove there account from Auth0. Returns ErrUserMissingId if email is empty string,
returns ErrInvalidEmail if the email address passed is not valid, returns ErrUserDeleteFailed if the DeletedCount is less than 1, and returns nil otherwise
*/
func DeleteUser(email string) error {
	_, err := GetUser(email)
	if err != nil {
		return err
	}

	var mongoDatabase = mtgContext.GetDatabase()

	_, valid := mongoDatabase.Delete("user", bson.M{"email": email})
	if !valid {
		return sdkErrors.ErrUserDeleteFailed
	}

	return nil
}

/*
RegisterUser Register a new user with Auth0 and store there user model within the MongoDB database
*/
func RegisterUser(username string, email string, password string) (*userModel.User, error) {
	ret := &userModel.User{
		Username: username,
		Email:    email,
		Stats:    &userModel.UserStatistics{},
	}

	if !validateEmail(email) {
		return ret, sdkErrors.ErrInvalidEmail
	}

	if len(password) < 12 {
		return ret, sdkErrors.ErrInvalidPasswordLength
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
		return ret, sdkErrors.ErrFailedToRegisterUser
	}

	ret.Auth0Id = userResp.ID

	err = NewUser(ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

/*
LoginUser Log a user in with there email address and password and return back an oauth.TokenSet
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
		Scope:    viper.GetString("auth0.scope"),
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
DeactivateUser Completely removes the requested user account, both from Auth0 and from MongoDB
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
ResetUserPassword Send a reset password email to a specified user account.
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
