package deck

import (
	"github.com/stevezaluk/mtgjson-sdk/context"

	deckModel "github.com/stevezaluk/mtgjson-models/deck"
	sdkErrors "github.com/stevezaluk/mtgjson-models/errors"
	"go.mongodb.org/mongo-driver/bson"
)

/*
ReplaceDeck Replace the entire deck in the database with the deck model
passed in the parameter. Returns ErrDeckUpdateFailed if the deck
cannot be located
*/
func ReplaceDeck(deck *deckModel.Deck) error {
	var database = context.GetDatabase()

	results := database.Replace("deck", bson.M{"code": deck.Code}, &deck)
	if results == nil {
		return sdkErrors.ErrDeckUpdateFailed
	}

	return nil
}

/*
DeleteDeck Remove a deck from the MongoDB database using the code passed in the
parameter. Returns ErrNoDeck if the deck does not exist. Returns
ErrDeckDeleteFailed if the deleted count does not equal 1
*/
func DeleteDeck(code string) any {
	var database = context.GetDatabase()

	query := bson.M{"code": code}
	result := database.Delete("deck", query)
	if result == nil {
		return sdkErrors.ErrNoDeck
	}

	if result.DeletedCount != 1 {
		return sdkErrors.ErrDeckDeleteFailed
	}

	return result
}

/*
GetDeck Fetch a deck from the MongoDB database using the code passed in the parameter. Returns
ErrNoDeck if the deck does not exist or cannot be located
*/
func GetDeck(code string) (*deckModel.Deck, error) {
	var result *deckModel.Deck

	var database = context.GetDatabase()

	query := bson.M{"code": code}
	results := database.Find("deck", query, &result)
	if results == nil {
		return result, sdkErrors.ErrNoDeck
	}

	return result, nil
}

/*
IndexDecks Returns all decks in the database unmarshalled as deck models. The limit parameter
will be passed directly to the database query to limit the number of models returned
*/
func IndexDecks(limit int64) ([]*deckModel.Deck, error) {
	var result []*deckModel.Deck

	var database = context.GetDatabase()

	results := database.Index("deck", limit, &result)
	if results == nil {
		return result, sdkErrors.ErrNoDecks
	}

	return result, nil
}

/*
NewDeck Insert a new deck in the form of a model into the MongoDB database. The deck model must have a
valid name and deck code, additionally the deck cannot already exist under the same deck code
*/
func NewDeck(deck *deckModel.Deck) error {
	if deck.Name == "" || deck.Code == "" {
		return sdkErrors.ErrDeckMissingId
	}

	if deck.ContentIds == nil {
		return sdkErrors.ErrDeckMissingId
	}

	_, err := GetDeck(deck.Code)
	if err != sdkErrors.ErrNoDeck {
		return sdkErrors.ErrDeckAlreadyExists
	}

	var database = context.GetDatabase()

	database.Insert("deck", &deck)

	return nil
}
