package card

import (
	"github.com/stevezaluk/mtgjson-sdk/context"
	"regexp"

	"github.com/stevezaluk/mtgjson-models/card"
	"github.com/stevezaluk/mtgjson-models/errors"
	"go.mongodb.org/mongo-driver/bson"
)

/*
ValidateUUID - Ensure that the passed UUID is valid

Paremeters:
uuid (string) - The UUID you want to validate

Returns:
ret (bool) - True if the UUID is valid, false if it is not
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
ValidateCards - Ensure a list of cards both exist and are valid UUID's

Paremeters:
uuids (array[string]) - A list of mtgjsonV4 UUID's to validate

Returns:
result (bool) - True if all cards passed validation, False if they didnt
invalidCards (array[string]) - Values that are not valid UUID's
noExistCards (array[string]) - Cards that do not exist in Mongo
*/
func ValidateCards(uuids []string) (bool, []string, []string) {
	var invalidCards []string // cards that failed UUID validation
	var noExistCards []string // cards that do not exist in Mongo
	var result = true

	for _, uuid := range uuids {
		_, err := GetCard(uuid)
		if err == errors.ErrNoCard {
			result = false
			noExistCards = append(noExistCards, uuid)
		} else if err == errors.ErrInvalidUUID {
			result = false
			invalidCards = append(invalidCards, uuid)
		}
	}

	return result, invalidCards, noExistCards
}

/*
GetCards - Takes a list of UUID's and returns card models for them

Paramters:
cards (slice[string]) - A list of UUID's you want card models for

Returns
ret (slice[card.Card]) - A list of card models
*/
func GetCards(cards []string) []card.Card {
	var ret []card.Card
	for i := 0; i < len(cards); i++ {
		uuid := cards[i]

		card, err := GetCard(uuid)
		if err != nil {
			continue
		}

		ret = append(ret, card)
	}

	return ret
}

/*
GetCard - Fetch a card model for a UUID

Parameters:
uuid (string) - The UUID you want a card model for

Returns
result (card.Card) - The card that was found
errors.ErrInvalidUUID - If the UUID is not valid
errors.ErrNoCard - If the card is not found
*/
func GetCard(uuid string) (card.Card, error) {
	var result card.Card

	if !ValidateUUID(uuid) {
		return result, errors.ErrInvalidUUID
	}

	var database = context.GetDatabase()

	query := bson.M{"identifiers.mtgjsonv4id": uuid}
	results := database.Find("card", query, &result)
	if results == nil {
		return result, errors.ErrNoCard
	}

	return result, nil
}

/*
NewCard - Create a new card from an existing card model

Parameters:
card (card.Card) - The card model you want to insert

Returns:
error.ErrCardMissingId - If the name or cardId is missing
errors.ErrCardAlreadyExists - If there is a mtgjsonV4Id conflict

TODO: Need better card validation here
*/
func NewCard(card card.Card) error {
	cardId := card.Identifiers.MTGJsonV4Id
	if card.Name == "" || cardId == "" {
		return errors.ErrCardMissingId
	}

	_, err := GetCard(cardId)
	if err != errors.ErrNoCard {
		return errors.ErrCardAlreadyExist
	}

	var database = context.GetDatabase()
	database.Insert("card", &card)

	return nil
}

/*
DeleteCard - Delete a card from the database

Parmeters:
uuid (string) - The mtgsonv4id you want to remove from the database

Returns:
errors.ErrNoCard - If the card does not exist
errors.ErrCardDeleteFailed - If the card was not deleted
*/
func DeleteCard(uuid string) error {
	var database = context.GetDatabase()

	query := bson.M{"identifiers.mtgjsonv4id": uuid}
	result := database.Delete("card", query)
	if result == nil {
		return errors.ErrNoCard
	}

	if result.DeletedCount != 1 {
		return errors.ErrCardDeleteFailed
	}

	return nil
}

/*
IndexCards - Return all cards from the database

Parameters:
limit (int64) - Limit the ammount of cards you want returned

Returns:
result (slice[card.Card]) - The results of the operation
errors.ErrNoCards - If the database has no cards
*/
func IndexCards(limit int64) ([]card.Card, error) {
	var result []card.Card

	var database = context.GetDatabase()

	results := database.Index("card", limit, &result)
	if results == nil {
		return result, errors.ErrNoCards
	}

	return result, nil

}
