package set

import (
	"github.com/stevezaluk/mtgjson-sdk/context"

	sdkErrors "github.com/stevezaluk/mtgjson-models/errors"
	"github.com/stevezaluk/mtgjson-models/set"
	"go.mongodb.org/mongo-driver/bson"
)

/*
GetSet Takes a single string representing a set code and returns a set model for the set.
Returns ErrNoSet if the set does not exist, or cannot be located
*/
func GetSet(code string) (*set.Set, error) {
	var ret *set.Set
	var database = context.GetDatabase()

	err := database.Find("set", bson.M{"code": code}, &ret)
	if !err {
		return ret, sdkErrors.ErrNoSet
	}

	return ret, nil
}

/*
IndexSets Returns all sets in the database unmarshalled as card models. The limit parameter
will be passed directly to the database query to limit the number of models returned
*/
func IndexSets(limit int64) ([]*set.Set, error) {
	var ret []*set.Set
	var database = context.GetDatabase()

	err := database.Index("set", limit, ret)
	if !err {
		return ret, sdkErrors.ErrNoSet
	}

	return ret, nil
}
