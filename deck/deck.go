package deck

import (
	"errors"
	cardModel "github.com/stevezaluk/mtgjson-models/card"
	"github.com/stevezaluk/mtgjson-models/meta"
	"github.com/stevezaluk/mtgjson-sdk/card"
	"github.com/stevezaluk/mtgjson-sdk/server"
	"github.com/stevezaluk/mtgjson-sdk/user"
	"github.com/stevezaluk/mtgjson-sdk/util"

	"slices"

	deckModel "github.com/stevezaluk/mtgjson-models/deck"
	sdkErrors "github.com/stevezaluk/mtgjson-models/errors"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	BoardMainboard = "mainBoard"
	BoardSideboard = "sideBoard"
	BoardCommander = "commander"
)

/*
ReplaceDeck Replace the entire deck in the database with the deck model
passed in the parameter. Returns ErrDeckUpdateFailed if the deck
cannot be located
*/
func ReplaceDeck(database *server.Database, deck *deckModel.Deck) error {
	_, err := database.Replace("deck", bson.M{"code": deck.Code}, &deck)
	if !err {
		return sdkErrors.ErrDeckUpdateFailed
	}

	return nil
}

/*
DeleteDeck Remove a deck from the MongoDB database using the code passed in the
parameter. Returns ErrNoDeck if the deck does not exist. Returns
ErrDeckDeleteFailed if the deleted count does not equal 1
*/
func DeleteDeck(database *server.Database, code string, owner string) error {
	query := bson.M{"code": code}
	if owner != "" {
		query = bson.M{"code": code, "mtgjsonApiMeta.owner": owner}
	}

	result, err := database.Delete("deck", query)
	if !err {
		return sdkErrors.ErrNoDeck
	}

	if result.DeletedCount != 1 {
		return sdkErrors.ErrDeckDeleteFailed
	}

	return nil
}

/*
GetDeck Fetch a deck from the MongoDB database using the code passed in the parameter. Owner
is the email address of the user that you want to assign to the deck. If the string is empty
then it does not filter by user. Returns ErrNoDeck if the deck does not exist or cannot be located
*/
func GetDeck(database *server.Database, code string, owner string) (*deckModel.Deck, error) {
	var result *deckModel.Deck

	query := bson.M{"code": code}
	if owner != "" {
		query = bson.M{"code": code, "mtgjsonApiMeta.owner": owner}
	}

	err := database.Find("deck", query, &result)
	if !err {
		return result, sdkErrors.ErrNoDeck
	}

	return result, nil
}

/*
IndexDecks Returns all decks in the database unmarshalled as deck models. The limit parameter
will be passed directly to the database query to limit the number of models returned
*/
func IndexDecks(database *server.Database, limit int64) ([]*deckModel.Deck, error) {
	var result []*deckModel.Deck

	err := database.Index("deck", limit, &result)
	if !err {
		return result, sdkErrors.ErrNoDecks
	}

	return result, nil
}

/*
NewDeck Insert a new deck in the form of a model into the MongoDB database. The deck model must have a
valid name and deck code, additionally the deck cannot already exist under the same deck code. Owner is
the email address of the owner you want to assign the deck to. If the string is empty, it will be assigned
to the system user
*/
func NewDeck(database *server.Database, deck *deckModel.Deck, owner string) error {
	if deck.Name == "" || deck.Code == "" {
		return sdkErrors.ErrDeckMissingId
	}

	if owner == "" {
		owner = user.SystemUser
	}

	if owner != user.SystemUser {
		_, err := user.GetUser(database, owner)
		if err != nil {
			return err
		}
	}

	_, err := GetDeck(database, deck.Code, owner)
	if !errors.Is(err, sdkErrors.ErrNoDeck) {
		return sdkErrors.ErrDeckAlreadyExists
	}

	if deck.ContentIds == nil {
		deck.ContentIds = &deckModel.DeckContentIds{
			MainBoard: []string{},
			SideBoard: []string{},
			Commander: []string{},
		}
	}

	currentDate := util.CreateTimestampStr()
	if deck.ReleaseDate == "" {
		deck.ReleaseDate = currentDate
	}

	deck.MtgjsonApiMeta = &meta.MTGJSONAPIMeta{
		Owner:        owner,
		Type:         "Deck",
		CreationDate: currentDate,
		ModifiedDate: currentDate,
	}

	database.Insert("deck", &deck)

	return nil
}
