package set

import (
	"github.com/stevezaluk/mtgjson-sdk/context"

	"github.com/stevezaluk/mtgjson-models/errors"
	"github.com/stevezaluk/mtgjson-models/set"
	"go.mongodb.org/mongo-driver/bson"
)

func GetSet(code string) (set.Set, error) {
	var ret set.Set
	var database = context.GetDatabase()

	results := database.Find("set", bson.M{"code": code}, &ret)
	if results == nil {
		return ret, errors.ErrNoSet
	}

	return ret, nil
}

func IndexSets(limit int64) ([]set.Set, error) {
	var ret []set.Set
	var database = context.GetDatabase()

	results := database.Index("set", limit, ret)
	if results == nil {
		return ret, errors.ErrNoSet
	}

	return ret, nil
}
