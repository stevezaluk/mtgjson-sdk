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

const (
	SystemUser = "system"
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
func GetUser(ctx *mtgContext.ServerContext, email string) (*userModel.User, error) {
	var result *userModel.User

	if email == "" {
		return nil, sdkErrors.ErrUserMissingId
	}

	if !validateEmail(email) {
		return nil, sdkErrors.ErrInvalidEmail
	}

	query := bson.M{"email": email}
	err := ctx.Database().Find("user", query, &result)
	if !err {
		return nil, sdkErrors.ErrNoUser
	}

	return result, nil
}

/*
GetEmailFromToken Fetch a users email from an authentication token passed to them
*/
func GetEmailFromToken(ctx *mtgContext.ServerContext, token string) (string, error) {

	userInfo, err := ctx.AuthAPI().UserInfo(context.Background(), token)
	if err != nil {
		return "", err
	}

	return userInfo.Email, nil
}

/*
NewUser Insert the contents of a User model in the MongoDB database. Returns ErrUserMissingId if the Username, or Email is not present
Returns ErrUserAlreadyExist if a user already exists under this username
*/
func NewUser(ctx *mtgContext.ServerContext, user *userModel.User) error {
	if user.Username == "" || user.Email == "" || user.Auth0Id == "" {
		return sdkErrors.ErrUserMissingId
	}

	if !validateEmail(user.Email) {
		return sdkErrors.ErrInvalidEmail
	}

	_, err := GetUser(ctx, user.Email)
	if !errors.Is(err, sdkErrors.ErrNoUser) {
		return sdkErrors.ErrUserAlreadyExist
	}

	if len(user.OwnedCards) == 0 || user.OwnedCards == nil {
		user.OwnedCards = []string{}
	}

	if len(user.OwnedSets) == 0 || user.OwnedSets == nil {
		user.OwnedSets = []string{}
	}

	if len(user.OwnedDecks) == 0 || user.OwnedDecks == nil {
		user.OwnedDecks = []string{}
	}

	ctx.Database().Insert("user", &user)

	return nil
}

/*
IndexUsers List all users from the database, and return them in a slice. A limit can be provided to ensure that too many objects
don't get returned
*/
func IndexUsers(ctx *mtgContext.ServerContext, limit int64) ([]*user.User, error) {
	var result []*user.User

	err := ctx.Database().Index("user", limit, &result)
	if !err {
		return nil, sdkErrors.ErrNoUser
	}

	return result, nil
}

/*
DeleteUser Removes the requested users account from the MongoDB database. Does not remove there account from Auth0. Returns ErrUserMissingId if email is empty string,
returns ErrInvalidEmail if the email address passed is not valid, returns ErrUserDeleteFailed if the DeletedCount is less than 1, and returns nil otherwise
*/
func DeleteUser(ctx *mtgContext.ServerContext, email string) error {
	_, err := GetUser(ctx, email)
	if err != nil {
		return err
	}

	_, valid := ctx.Database().Delete("user", bson.M{"email": email})
	if !valid {
		return sdkErrors.ErrUserDeleteFailed
	}

	return nil
}

/*
RegisterUser Register a new user with Auth0 and store there user model within the MongoDB database
*/
func RegisterUser(ctx *mtgContext.ServerContext, username string, email string, password string) (*userModel.User, error) {
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

	userResp, err := ctx.AuthAPI().Database.Signup(context.Background(), userData)
	if err != nil {
		return ret, sdkErrors.ErrFailedToRegisterUser
	}

	ret.Auth0Id = userResp.ID

	err = NewUser(ctx, ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

/*
LoginUser Log a user in with there email address and password and return back an oauth.TokenSet
*/
func LoginUser(ctx *mtgContext.ServerContext, email string, password string) (*oauth.TokenSet, error) {
	_, err := GetUser(ctx, email)
	if err != nil {
		return nil, err
	}

	userData := oauth.LoginWithPasswordRequest{
		Username: email,
		Password: password,
		Audience: viper.GetString("auth0.audience"),
		Scope:    viper.GetString("auth0.scope"),
	}

	validateOpts := oauth.IDTokenValidationOptions{}

	token, err := ctx.AuthAPI().OAuth.LoginWithPassword(
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
func DeactivateUser(ctx *mtgContext.ServerContext, email string) error {
	user, err := GetUser(ctx, email)
	if err != nil {
		return err
	}

	err = DeleteUser(ctx, email)
	if err != nil {
		return err
	}

	userId := "auth0|" + user.Auth0Id

	err = ctx.AuthManagementAPI().User.Delete(context.TODO(), userId)
	if err != nil {
		return err
	}

	return nil
}

/*
ResetUserPassword Send a reset password email to a specified user account.
*/
func ResetUserPassword(ctx *mtgContext.ServerContext, email string) error {
	_, err := GetUser(ctx, email)
	if err != nil {
		return err
	}

	resetPwdRequest := database.ChangePasswordRequest{
		Email:      email,
		Connection: "Username-Password-Authentication",
	}

	_, err = ctx.AuthAPI().Database.ChangePassword(
		context.Background(),
		resetPwdRequest,
	)

	if err != nil {
		return err
	}

	return nil
}
