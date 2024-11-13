package user

import (
	"github.com/stevezaluk/mtgjson-models/errors"
	"github.com/stevezaluk/mtgjson-models/user"
	"github.com/stevezaluk/mtgjson-sdk/context"
	"go.mongodb.org/mongo-driver/bson"
)

/*
Fetch a user based on there username. Returns ErrNoUser if the user cannot be found
*/
func GetUser(username string) (user.User, error) {
	var result user.User

	var database = context.GetDatabase()

	query := bson.M{"username": username}
	results := database.Find("user", query, &result)
	if results == nil {
		return result, errors.ErrNoUser
	}

	return result, nil
}
