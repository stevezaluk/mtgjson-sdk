package deck

import (
	"github.com/stevezaluk/mtgjson-sdk/context"

	card_model "github.com/stevezaluk/mtgjson-models/card"
	deck_model "github.com/stevezaluk/mtgjson-models/deck"
	"github.com/stevezaluk/mtgjson-models/errors"
	card "github.com/stevezaluk/mtgjson-sdk/card"
	"go.mongodb.org/mongo-driver/bson"
)

/*
Iterate through all the cards in the deck's mainboard and return a list of
card models representing it
*/
func GetMainboard(deck deck_model.Deck) []card_model.Card {
	return card.GetCards(deck.Mainboard)
}

/*
Iterate through all the cards in the deck's sideboard and return a list of
card models representing it
*/
func GetSideboard(deck deck_model.Deck) []card_model.Card {
	return card.GetCards(deck.Sideboard)
}

/*
Iterate through all the cards in the deck's commander board and return a list of
card models representing it
*/
func GetCommanders(deck deck_model.Deck) []card_model.Card {
	return card.GetCards(deck.Commander)
}

/*
Replace the entire deck in the database with the deck model
passed in the parameter. Returns ErrDeckUpdateFailed if the deck
cannot be located
*/
func ReplaceDeck(deck deck_model.Deck) error {
	var database = context.GetDatabase()

	results := database.Replace("deck", bson.M{"code": deck.Code}, &deck)
	if results == nil {
		return errors.ErrDeckUpdateFailed
	}

	return nil
}

/*
Remove a deck from the MongoDB database using the code passed in the
parameter. Returns ErrNoDeck if the deck does not exist. Returns
ErrDeckDeleteFailed if the deleted count does not equal 1
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
Fetch a deck from the MongoDB database using the code passed in the parameter. Returns
ErrNoDeck if the deck does not exist or cannot be located
*/
func GetDeck(code string) (deck_model.Deck, error) {
	var result deck_model.Deck

	var database = context.GetDatabase()

	query := bson.M{"code": code}
	results := database.Find("deck", query, &result)
	if results == nil {
		return result, errors.ErrNoDeck
	}

	return result, nil
}

/*
Updates the `contents` field in the passed deck model with the requested cards
*/
func GetDeckContents(deck *deck_model.Deck) {
	var mainBoard = deck.GetBoard(deck_model.MAINBOARD)
	*mainBoard = append(*mainBoard, GetMainboard(*deck)...)

	var sideBoard = deck.GetBoard(deck_model.SIDEBOARD)
	*sideBoard = append(*sideBoard, GetSideboard(*deck)...)

	var commanderBoard = deck.GetBoard(deck_model.COMMANDER)
	*commanderBoard = append(*commanderBoard, GetCommanders(*deck)...)
}

/*
Returns all decks in the database unmarshalled as deck models. The limit parameter
will be passed directly to the database query to limit the number of models returned
*/
func IndexDecks(limit int64) ([]deck_model.Deck, error) {
	var result []deck_model.Deck

	var database = context.GetDatabase()

	results := database.Index("deck", limit, &result)
	if results == nil {
		return result, errors.ErrNoDecks
	}

	return result, nil
}

/*
Insert a new deck in the form of a model into the MongoDB database. The deck model must have a
valid name and deck code, additionally the deck cannot already exist under the same deck code
*/
func NewDeck(deck deck_model.Deck) error {
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
