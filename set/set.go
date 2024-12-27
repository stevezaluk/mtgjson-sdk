package set

import (
	"errors"
	"github.com/stevezaluk/mtgjson-models/meta"
	"github.com/stevezaluk/mtgjson-sdk/card"
	"github.com/stevezaluk/mtgjson-sdk/context"
	"github.com/stevezaluk/mtgjson-sdk/user"
	"github.com/stevezaluk/mtgjson-sdk/util"

	sdkErrors "github.com/stevezaluk/mtgjson-models/errors"
	"github.com/stevezaluk/mtgjson-models/set"
	"go.mongodb.org/mongo-driver/bson"
)

/*
ReplaceSet Replace the entire set in the database with the model passed in the parameter.
Returns ErrSetUpdateFailed if the set cannot be located
*/
func ReplaceSet(set *set.Set) error {
	var database = context.GetDatabase()

	_, err := database.Replace("set", bson.M{"code": set.Code}, &set)
	if !err {
		return sdkErrors.ErrSetUpdateFailed
	}

	return nil
}

/*
GetSet Takes a single string representing a set code and returns a set model for the set.
Returns ErrNoSet if the set does not exist, or cannot be located
*/
func GetSet(code string, owner string) (*set.Set, error) {
	var ret *set.Set
	var database = context.GetDatabase()

	query := bson.M{"code": code}
	if owner != "" {
		query = bson.M{"code": code, "mtgjsonApiMeta.owner": owner}
	}

	err := database.Find("set", query, &ret)
	if !err {
		return ret, sdkErrors.ErrNoSet
	}

	return ret, nil
}

/*
NewSet Insert a new set in the form of a model into the MongoDB database. The set model must have a
valid name and set code, additionally the set cannot already exist under the same set code. Owner is
the email address of the owner you want to assign the deck to. If the string is empty (i.e. == ""), it
will be assigned to the system user
*/
func NewSet(set *set.Set, owner string) error {
	if set.Name == "" || set.Code == "" {
		return sdkErrors.ErrSetMissingId
	}

	if owner == "" {
		owner = user.SystemUser
	}

	if owner != user.SystemUser {
		_, err := user.GetUser(owner)
		if err != nil {
			return err
		}
	}

	var database = context.GetDatabase()

	_, err := GetSet(set.Code, owner)
	if !errors.Is(err, sdkErrors.ErrNoSet) {
		return sdkErrors.ErrSetAlreadyExists
	}

	currentDate := util.CreateTimestampStr()
	if set.ReleaseDate == "" {
		set.ReleaseDate = currentDate
	}

	set.MtgjsonApiMeta = &meta.MTGJSONAPIMeta{
		Owner:        owner,
		Type:         "Set",
		CreationDate: currentDate,
		ModifiedDate: currentDate,
	}

	database.Insert("set", &set)

	return nil
}

/*
AddCards Update the contentIds in the set model passed with new cards.
This should probably perform card validation in the future
*/
func AddCards(set *set.Set, newCards []string) error {
	set.ContentIds = append(set.ContentIds, newCards...)

	set.MtgjsonApiMeta.ModifiedDate = util.CreateTimestampStr() // need better error checking here

	err := ReplaceSet(set)
	if err != nil {
		return err
	}

	return nil
}

func GetSetContents(set *set.Set) error {
	if set.ContentIds == nil || len(set.ContentIds) == 0 {
		return sdkErrors.ErrSetMissingId
	}

	contents, err := card.GetCards(set.ContentIds)
	if err != nil {
		return err
	}

	set.Contents = contents

	return nil
}

/*
DeleteSet Remove a set from the MongoDB database using the code passed in the parameter.
Returns ErrNoSet if the set does not exist. Returns ErrSetDeleteFailed if the deleted count
does not equal 1
*/
func DeleteSet(code string, owner string) error {
	var database = context.GetDatabase()

	query := bson.M{"code": code}
	if owner != "" {
		query = bson.M{"code": code, "mtgjsonApiMeta.owner": owner}
	}

	result, err := database.Delete("set", query)
	if !err {
		return sdkErrors.ErrNoSet
	}

	if result.DeletedCount != 1 {
		return sdkErrors.ErrSetDeleteFailed
	}

	return nil

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
