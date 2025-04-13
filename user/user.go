package user

import (
	"errors"
	"os/user"
	"regexp"

	sdkErrors "github.com/stevezaluk/mtgjson-models/errors"
	userModel "github.com/stevezaluk/mtgjson-models/user"
	"github.com/stevezaluk/mtgjson-sdk/server"
	"go.mongodb.org/mongo-driver/bson"
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
func GetUser(database *server.Database, email string) (*userModel.User, error) {
	var result *userModel.User

	if email == "" {
		return nil, sdkErrors.ErrUserMissingId
	}

	if !validateEmail(email) {
		return nil, sdkErrors.ErrInvalidEmail
	}

	query := bson.M{"email": email}
	err := database.Find("user", query, &result)
	if !err {
		return nil, sdkErrors.ErrNoUser
	}

	return result, nil
}

/*
NewUser Insert the contents of a User model in the MongoDB database. Returns ErrUserMissingId if the Username, or Email is not present
Returns ErrUserAlreadyExist if a user already exists under this username
*/
func NewUser(database *server.Database, user *userModel.User) error {
	if user.Username == "" || user.Email == "" || user.Auth0Id == "" {
		return sdkErrors.ErrUserMissingId
	}

	if !validateEmail(user.Email) {
		return sdkErrors.ErrInvalidEmail
	}

	_, err := GetUser(database, user.Email)
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

	database.Insert("user", &user)

	return nil
}

/*
IndexUsers List all users from the database, and return them in a slice. A limit can be provided to ensure that too many objects
don't get returned
*/
func IndexUsers(database *server.Database, limit int64) ([]*user.User, error) {
	var result []*user.User

	err := database.Index("user", limit, &result)
	if !err {
		return nil, sdkErrors.ErrNoUser
	}

	return result, nil
}

/*
DeleteUser Removes the requested users account from the MongoDB database. Does not remove there account from Auth0. Returns ErrUserMissingId if email is empty string,
returns ErrInvalidEmail if the email address passed is not valid, returns ErrUserDeleteFailed if the DeletedCount is less than 1, and returns nil otherwise
*/
func DeleteUser(database *server.Database, email string) error {
	_, err := GetUser(database, email)
	if err != nil {
		return err
	}

	_, valid := database.Delete("user", bson.M{"email": email})
	if !valid {
		return sdkErrors.ErrUserDeleteFailed
	}

	return nil
}
