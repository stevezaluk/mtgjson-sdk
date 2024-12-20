package card

import (
	"errors"
	"github.com/stevezaluk/mtgjson-sdk/context"
	"regexp"

	"github.com/stevezaluk/mtgjson-models/card"
	sdkErrors "github.com/stevezaluk/mtgjson-models/errors"
	"go.mongodb.org/mongo-driver/bson"
)

/*
ValidateUUID Validates that the string passed in the argument is a Version 4 UUID. Returns true
if validation passes, false otherwise
*/
func ValidateUUID(uuid string) bool {
	var ret = false
	var pattern = `^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`

	re := regexp.MustCompile(pattern)
	if re.MatchString(uuid) {
		ret = true
	}

	return ret
}

/*
ValidateCards Takes a list of strings representing MTGJSONv4 UUID's and ensures that they are both
valid and exist. Returns 3 variables a boolean and two lists of strings. The boolean
can be used as a general determination if the validation succeeded
*/
func ValidateCards(uuids []string) (bool, []string, []string) {
	var invalidCards []string // cards that failed UUID validation
	var noExistCards []string // cards that do not exist in Mongo
	var result = true

	for _, uuid := range uuids {
		_, err := GetCard(uuid)
		if errors.Is(err, sdkErrors.ErrNoCard) {
			result = false
			noExistCards = append(noExistCards, uuid)
		} else if errors.Is(err, sdkErrors.ErrInvalidUUID) {
			result = false
			invalidCards = append(invalidCards, uuid)
		}
	}

	return result, invalidCards, noExistCards
}

/*
GetCards Takes a list of strings representing MTGJSONv4 UUID's and returns a list of card models
representing them. Change this to process all cards in a single database call
*/
func GetCards(cards []string) []*card.CardSet {
	var ret []*card.CardSet
	for i := 0; i < len(cards); i++ {
		uuid := cards[i]

		cardModel, err := GetCard(uuid)
		if err != nil {
			continue
		}

		ret = append(ret, cardModel)
	}

	return ret
}

/*
GetCard Takes a single string representing an MTGJSONv4 UUID and return a card model
for it
*/
func GetCard(uuid string) (*card.CardSet, error) {
	var result card.CardSet

	if !ValidateUUID(uuid) {
		return &result, sdkErrors.ErrInvalidUUID
	}

	var database = context.GetDatabase()

	query := bson.M{"identifiers.mtgjsonV4Id": uuid}
	results := database.Find("card", query, &result)
	if results == nil {
		return &result, sdkErrors.ErrNoCard
	}

	return &result, nil
}

/*
NewCard Insert a new card in the form of a model into the MongoDB database. The card model must have a
valid name and MTGJSONv4 ID, additionally, the card cannot already exist under the same ID
*/
func NewCard(card *card.CardSet) error {
	cardId := card.Identifiers.MtgjsonV4Id
	if card.Name == "" || cardId == "" {
		return sdkErrors.ErrCardMissingId
	}

	_, err := GetCard(cardId)
	if !errors.Is(err, sdkErrors.ErrNoCard) {
		return sdkErrors.ErrCardAlreadyExist
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
func DeleteCard(uuid string) error {
	var database = context.GetDatabase()

	query := bson.M{"identifiers.mtgjsonV4Id": uuid}
	result := database.Delete("card", query)
	if result == nil {
		return sdkErrors.ErrNoCard
	}

	if result.DeletedCount != 1 {
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

	results := database.Index("card", limit, &result)
	if results == nil {
		return result, sdkErrors.ErrNoCards
	}

	return result, nil

}
