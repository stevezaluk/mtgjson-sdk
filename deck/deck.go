package deck

import (
	"github.com/stevezaluk/mtgjson-sdk/context"

	card_model "github.com/stevezaluk/mtgjson-models/card"
	"github.com/stevezaluk/mtgjson-models/deck"
	"github.com/stevezaluk/mtgjson-models/errors"
	"go.mongodb.org/mongo-driver/bson"
	card "github.com/stevezaluk/mtgjson-sdk/card"
)

/*
FetchMainboard - Iterate through the UUID's in the main board and return card models

Parameters:
None

Returns
slice[card.Card] - The results
*/
func FetchMainboard(deck deck.Deck) []card_model.Card {
	return card.GetCards(deck.MainBoard)
}

/*
FetchSideboard - Iterate through the UUID's in the side board and return card models

Parameters:
None

Returns
slice[card.Card] - The results
*/
func FetchSideboard(deck deck.Deck) []card_model.Card {
	return card.GetCards(deck.SideBoard)
}

/*
FetchCommander - Iterate through the UUID's in the commander board and return card models

Parameters:
None

Returns
slice[card.Card] - The results
*/
func FetchCommander(deck deck.Deck) []card_model.Card {
	return card.GetCards(deck.Commander)
}

/*
UpdateDeck - Replace the deck in the database

Parameters:
None

Returns:
error.ErrDeckUpdateFailed - If database.Replace returns an error
*/
func UpdateDeck(deck deck.Deck) error {
	var database = context.GetDatabase()

	results := database.Replace("deck", bson.M{"code": deck.Code}, &deck)
	if results == nil {
		return errors.ErrDeckUpdateFailed
	}

	return nil
}

/*
DeleteDeck - Delete the deck from the database

Parameters:
None

Returns:
errors.ErrNoDeck - If the deck does not exist
errors.ErrDeckDeleteFailed - If the mongo results structure doesn't show any deleted results
*/
func DeleteDeck(code string) any {
	var database = context.GetDatabase()

	query := bson.M{"code": code}
	result := database.Delete("deck", query)
	if result == nil {
		return errors.ErrNoDeck
	}

	if result.DeletedCount != 1 {
		return errors.ErrDeckDeleteFailed
	}

	return result
}

/*
GetDeck - Fetch a deck model and from a deck code

Parameters:
code (string) - The deck code

Returns
Deck (deck.Deck) - A deck model
errors.ErrNoDeck - If the deck does not exist
*/
func GetDeck(code string) (deck.Deck, error) {
	var result deck.Deck

	var database = context.GetDatabase()

	query := bson.M{"code": code}
	results := database.Find("deck", query, &result)
	if results == nil {
		return result, errors.ErrNoDeck
	}

	return result, nil
}

/*
GetDecks - Fetch all decks available in the database

Parameters:
limit (int64) - Limit the ammount of results you want

Returns:
result (slice[deck.Deck]) - The results
errors.ErrNoDecks - If no decks exist in the database
*/
func GetDecks(limit int64) ([]deck.Deck, error) {
	var result []deck.Deck

	var database = context.GetDatabase()

	results := database.Index("deck", limit, &result)
	if results == nil {
		return result, errors.ErrNoDecks
	}

	return result, nil
}

/*
NewDeck - Create a new deck from a deck model

Parameters:
errors.ErrDeskMissingId - If the deck passed in the parameter does not have a valid name or code
errors.ErrDeckAlreadyExists - If the deck already exists under the same code
*/
func NewDeck(deck deck.Deck) error {
	if deck.Name == "" || deck.Code == "" {
		return errors.ErrDeckMissingId
	}

	_, valid := GetDeck(deck.Code)
	if valid != errors.ErrNoDeck {
		return errors.ErrDeckAlreadyExists
	}

	var database = context.GetDatabase()

	database.Insert("deck", &deck)

	return nil
}
