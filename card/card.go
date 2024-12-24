package card

import (
	"errors"
	"github.com/stevezaluk/mtgjson-models/meta"
	"github.com/stevezaluk/mtgjson-sdk/context"
	"github.com/stevezaluk/mtgjson-sdk/user"
	"github.com/stevezaluk/mtgjson-sdk/util"
	"regexp"
	"slices"

	"github.com/stevezaluk/mtgjson-models/card"
	sdkErrors "github.com/stevezaluk/mtgjson-models/errors"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	UUIDRegexPattern = `^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`
)

var (
	UUIDRegex = regexp.MustCompile(UUIDRegexPattern)
)

/*
ValidateUUID Validates that the string passed in the argument is a Version 4 UUID. Returns true
if validation passes, false otherwise
*/
func ValidateUUID(uuid string) bool {
	var ret = false

	if UUIDRegex.MatchString(uuid) {
		ret = true
	}

	return ret
}

/*
ValidateCards Takes a list of strings representing MTGJSONv4 UUID's and ensures that they are both
valid and exist. Returns 3 variables: an error, and two lists of strings.
*/
func ValidateCards(uuids []string) (error, []string, []string) {
	var invalidCards []string // cards that failed UUID validation
	var noExistCards []string // cards that do not exist in Mongo

	cards, err := GetCards(uuids)
	if err != nil {
		return err, invalidCards, noExistCards
	}

	cardUuids := ExtractCardIds(cards)

	for _, uuid := range uuids {
		isValidUUID := ValidateUUID(uuid)
		if !isValidUUID {
			invalidCards = append(invalidCards, uuid)
			continue
		}

		if !slices.Contains(cardUuids, uuid) {
			noExistCards = append(noExistCards, uuid)
		}
	}

	return nil, invalidCards, noExistCards
}

/*
ExtractCardIds Take a list of CardSet models and return the UUID's from them
*/
func ExtractCardIds(cards []*card.CardSet) []string {
	var ret []string

	for _, card := range cards {
		if card.Identifiers == nil {
			continue
		}

		ret = append(ret, card.Identifiers.MtgjsonV4Id)
	}

	return ret
}

/*
GetCards Takes a list of strings representing MTGJSONv4 UUID's and returns a list of card models
representing them. Change this to process all cards in a single database call
*/
func GetCards(cards []string) ([]*card.CardSet, error) {
	var ret []*card.CardSet

	var database = context.GetDatabase()

	err := database.FindMultiple("card", "identifiers.mtgjsonV4Id", cards, &ret)
	if !err {
		return nil, sdkErrors.ErrNoCards
	}

	return ret, nil
}

/*
GetCard Takes a single string representing an MTGJSONv4 UUID and return a card model
for it
*/
func GetCard(uuid string, owner string) (*card.CardSet, error) {
	var result card.CardSet

	if !ValidateUUID(uuid) {
		return &result, sdkErrors.ErrInvalidUUID
	}

	var database = context.GetDatabase()

	query := bson.M{"identifiers.mtgjsonV4Id": uuid}
	if owner != "" {
		owner = "system"
	}

	err := database.Find("card", query, &result)
	if !err {
		return nil, sdkErrors.ErrNoCard
	}

	return &result, nil
}

/*
NewCard Insert a new card in the form of a model into the MongoDB database. The card model must have a
valid name and MTGJSONv4 ID, additionally, the card cannot already exist under the same ID
*/
func NewCard(card *card.CardSet, owner string) error {
	if card.Identifiers == nil {
		return sdkErrors.ErrCardMissingId
	}

	cardId := card.Identifiers.MtgjsonV4Id
	if card.Name == "" || cardId == "" {
		return sdkErrors.ErrCardMissingId
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

	_, err := GetCard(cardId, owner)
	if !errors.Is(err, sdkErrors.ErrNoCard) {
		return sdkErrors.ErrCardAlreadyExist
	}

	currentDate := util.CreateTimestampStr()
	card.MtgjsonApiMeta = &meta.MTGJSONAPIMeta{
		Owner:        owner,
		Type:         "Card",
		Subtype:      "Set",
		CreationDate: currentDate,
		ModifiedDate: currentDate,
	}

	var database = context.GetDatabase()
	database.Insert("card", &card)

	return nil
}

/*
DeleteCard Remove a card from the MongoDB database. The UUID passed in the parameter must be a valid MTGJSONv4 ID.
ErrNoCard will be returned if no card exists under the passed UUID, and ErrCardDeleteFailed will be returned
if the deleted count does not equal 1
*/
func DeleteCard(uuid string, owner string) error {
	var database = context.GetDatabase()

	query := bson.M{"identifiers.mtgjsonV4Id": uuid}
	if owner != "" {
		query = bson.M{"identifiers.mtgjsonV4Id": uuid, "mtgjsonApiMeta.owner": owner}
	}
	result, err := database.Delete("card", query)
	if !err {
		return sdkErrors.ErrNoCard
	}

	if result.DeletedCount < 1 {
		return sdkErrors.ErrCardDeleteFailed
	}

	return nil
}

/*
IndexCards Returns all cards in the database unmarshalled as card models. The limit parameter
will be passed directly to the database query to limit the number of models returned
*/
func IndexCards(limit int64) ([]*card.CardSet, error) {
	var result []*card.CardSet

	var database = context.GetDatabase()

	err := database.Index("card", limit, &result)
	if !err {
		return nil, sdkErrors.ErrNoCards
	}

	return result, nil

}
