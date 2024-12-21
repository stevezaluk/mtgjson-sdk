package set

import (
	"github.com/stevezaluk/mtgjson-sdk/context"

	"github.com/stevezaluk/mtgjson-models/errors"
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

	results := database.Find("set", bson.M{"code": code}, &ret)
	if results == nil {
		return ret, errors.ErrNoSet
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

	results := database.Index("set", limit, ret)
	if results == nil {
		return ret, errors.ErrNoSet
	}

	return ret, nil
}
