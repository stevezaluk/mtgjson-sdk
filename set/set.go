package set

import (
	"errors"
	"github.com/stevezaluk/mtgjson-models/meta"
	"github.com/stevezaluk/mtgjson-sdk/card"
	"github.com/stevezaluk/mtgjson-sdk/context"
	"github.com/stevezaluk/mtgjson-sdk/user"
	"github.com/stevezaluk/mtgjson-sdk/util"
	"slices"

	sdkErrors "github.com/stevezaluk/mtgjson-models/errors"
	"github.com/stevezaluk/mtgjson-models/set"
	"go.mongodb.org/mongo-driver/bson"
)

/*
ReplaceSet Replace the entire set in the database with the model passed in the parameter.
Returns ErrSetUpdateFailed if the set cannot be located
*/
func ReplaceSet(ctx *context.ServerContext, set *set.Set) error {

	_, err := ctx.Database().Replace("set", bson.M{"code": set.Code}, &set)
	if !err {
		return sdkErrors.ErrSetUpdateFailed
	}

	return nil
}

/*
GetSet Takes a single string representing a set code and returns a set model for the set.
Returns ErrNoSet if the set does not exist, or cannot be located
*/
func GetSet(ctx *context.ServerContext, code string, owner string) (*set.Set, error) {
	var ret *set.Set

	query := bson.M{"code": code}
	if owner != "" {
		query = bson.M{"code": code, "mtgjsonApiMeta.owner": owner}
	}

	err := ctx.Database().Find("set", query, &ret)
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
func NewSet(ctx *context.ServerContext, set *set.Set, owner string) error {
	if set.Name == "" || set.Code == "" {
		return sdkErrors.ErrSetMissingId
	}

	if owner == "" {
		owner = user.SystemUser
	}

	if owner != user.SystemUser {
		_, err := user.GetUser(ctx, owner)
		if err != nil {
			return err
		}
	}

	_, err := GetSet(ctx, set.Code, owner)
	if !errors.Is(err, sdkErrors.ErrNoSet) {
		return sdkErrors.ErrSetAlreadyExists
	}

	if set.ContentIds == nil || len(set.ContentIds) == 0 {
		set.ContentIds = []string{}
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

	ctx.Database().Insert("set", &set)

	return nil
}

/*
AddCards Update the contentIds in the set model passed with new cards.
This should probably perform card validation in the future. This should also be updated
to allow multiples of cards to be added
*/
func AddCards(ctx *context.ServerContext, set *set.Set, newCards []string) error {
	if newCards == nil || len(newCards) == 0 {
		return nil // no new cards to add. returning nil here to not consume a database call
	}

	set.ContentIds = append(set.ContentIds, newCards...)

	if set.MtgjsonApiMeta == nil {
		return sdkErrors.ErrMissingMetaApi
	}

	set.MtgjsonApiMeta.ModifiedDate = util.CreateTimestampStr() // need better error checking here

	err := ReplaceSet(ctx, set)
	if err != nil {
		return err
	}

	return nil
}

/*
RemoveCards Update the contentIds in the set model with the cards to be removed in the
cards array. This should be updated to support removing multiples of one card at a time
*/
func RemoveCards(ctx *context.ServerContext, set *set.Set, cards []string) error {
	if cards == nil || len(cards) == 0 {
		return nil // no new cards to add. returning nil here to not consume a database call
	}

	for _, uuid := range cards {
		for index, value := range set.ContentIds {
			if value == uuid {
				set.ContentIds = slices.Delete(set.ContentIds, index, index+1)
			}
		}
	}

	if set.MtgjsonApiMeta == nil {
		return sdkErrors.ErrMissingMetaApi
	}

	set.MtgjsonApiMeta.ModifiedDate = util.CreateTimestampStr()

	err := ReplaceSet(ctx, set)
	if err != nil {
		return err
	}

	return nil
}

/*
GetSetContents Update the contents field of the set passed in the parameter using the GetCards
function. Consumes a single database call. If the contentIds field is nil or has a length of 0,
it will return nil and abort the call
*/
func GetSetContents(ctx *context.ServerContext, set *set.Set) error {
	if set.ContentIds == nil || len(set.ContentIds) == 0 {
		return nil // returning nil here to not consume a database call
	}

	contents, err := card.GetCards(ctx, set.ContentIds)
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
func DeleteSet(ctx *context.ServerContext, code string, owner string) error {

	query := bson.M{"code": code}
	if owner != "" {
		query = bson.M{"code": code, "mtgjsonApiMeta.owner": owner}
	}

	result, err := ctx.Database().Delete("set", query)
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
func IndexSets(ctx *context.ServerContext, limit int64) ([]*set.Set, error) {
	var ret []*set.Set

	err := ctx.Database().Index("set", limit, ret)
	if !err {
		return ret, sdkErrors.ErrNoSet
	}

	return ret, nil
}
