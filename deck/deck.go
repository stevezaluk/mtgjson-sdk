package deck

import (
	"errors"
	"github.com/stevezaluk/mtgjson-models/meta"
	"github.com/stevezaluk/mtgjson-sdk/card"
	"github.com/stevezaluk/mtgjson-sdk/server"
	"github.com/stevezaluk/mtgjson-sdk/user"
	"github.com/stevezaluk/mtgjson-sdk/util"
	"maps"
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
AllCardIds - Takes a deckModel.DeckContentIds structure and retuns a single slice of strings
representing all the cardIds across each board
*/
func AllCardIds(contents *deckModel.DeckContentIds) []string {
	var allIds []string

	allIds = append(slices.Collect(maps.Keys(contents.MainBoard)), slices.Collect(maps.Keys(contents.SideBoard))...)
	allIds = append(allIds, slices.Collect(maps.Keys(contents.Commander))...)

	return allIds
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

	if deck.Contents == nil {
		contents := &deckModel.DeckContentIds{
			MainBoard: map[string]int64{},
			SideBoard: map[string]int64{},
			Commander: map[string]int64{},
		}

		deck.Contents = contents
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

/*
GetDeckContents - Iterates through all the boards in a deck and fetches the card models for each of the cards.
First all the cardID's across all boards are appended to a single list and a single database call is
consumed to fetch them down. Then they are iterated over and each board is checked for the ID, if it is found
then it is added its respective board as a deckModel.DeckContentEntry structure
*/
func GetDeckContents(database *server.Database, deck *deckModel.Deck) (*deckModel.DeckContents, error) {
	if deck.Contents == nil {
		return nil, sdkErrors.ErrDeckMissingContentIds
	}

	if deck.Code == "" || deck.MtgjsonApiMeta.Owner == "" {
		return nil, sdkErrors.ErrDeckMissingId
	}

	ret := &deckModel.DeckContents{
		MainBoard: map[string]*deckModel.DeckContentEntry{},
		SideBoard: map[string]*deckModel.DeckContentEntry{},
		Commander: map[string]*deckModel.DeckContentEntry{},
	}

	allCards, err := card.GetCards(database, AllCardIds(deck.Contents))
	if err != nil {
		return nil, err
	}

	for _, requestedCard := range allCards {
		id := requestedCard.Identifiers.MtgjsonV4Id

		quantity := deck.Contents.MainBoard[id]
		if quantity != 0 { // cardId exists
			ret.MainBoard[id] = &deckModel.DeckContentEntry{
				Quantity: quantity,
				Card:     requestedCard,
			}
		}

		quantity = deck.Contents.SideBoard[id]
		if quantity != 0 { // cardId exists
			ret.SideBoard[id] = &deckModel.DeckContentEntry{
				Quantity: quantity,
				Card:     requestedCard,
			}
		}

		quantity = deck.Contents.Commander[id]
		if quantity != 0 { // cardId exists
			ret.Commander[id] = &deckModel.DeckContentEntry{
				Quantity: quantity,
				Card:     requestedCard,
			}
		}
	}

	return ret, nil
}

/*
AddCards - Add cards to a deck within the database. Deck must have a Deck Code associated with it or it will
error out. Does not validate cards
*/
func AddCards(database *server.Database, deck *deckModel.Deck, contents *deckModel.DeckContentIds) error {
	if deck.Code == "" {
		return sdkErrors.ErrDeckMissingId
	}

	if deck.Contents == nil {
		return sdkErrors.ErrDeckMissingContentIds
	}

	for id, quantity := range contents.MainBoard {
		check := deck.Contents.MainBoard[id]
		if check != 0 {
			deck.Contents.MainBoard[id] = quantity + check
		} else {
			deck.Contents.MainBoard[id] = quantity
		}
	}

	for id, quantity := range contents.SideBoard {
		check := deck.Contents.SideBoard[id]
		if check != 0 {
			deck.Contents.SideBoard[id] = quantity + check
		} else {
			deck.Contents.SideBoard[id] = quantity
		}
	}

	for id, quantity := range contents.Commander {
		check := deck.Contents.Commander[id]
		if check != 0 {
			deck.Contents.Commander[id] = quantity + check
		} else {
			deck.Contents.Commander[id] = quantity
		}
	}

	deck.MtgjsonApiMeta.ModifiedDate = util.CreateTimestampStr()

	// this is really in-efficient and should be changed
	err := ReplaceDeck(database, deck)
	if err != nil {
		return err
	}

	return nil
}

/*
RemoveCards - Remove cards from a specified board. Does not validate cards
*/
func RemoveCards(database *server.Database, deck *deckModel.Deck, contents *deckModel.DeckContentIds) error {
	if deck.Code == "" {
		return sdkErrors.ErrDeckMissingId
	}

	if deck.Contents == nil {
		return sdkErrors.ErrDeckMissingContentIds
	}

	for id, quantity := range contents.MainBoard {
		check := deck.Contents.MainBoard[id]
		if check != 0 {
			deck.Contents.MainBoard[id] = check - quantity
		}

		if deck.Contents.MainBoard[id] == 0 {
			delete(deck.Contents.MainBoard, id)
		}
	}

	for id, quantity := range contents.SideBoard {
		check := deck.Contents.SideBoard[id]
		if check != 0 {
			deck.Contents.SideBoard[id] = check - quantity
		}

		if deck.Contents.SideBoard[id] == 0 {
			delete(deck.Contents.SideBoard, id)
		}
	}

	for id, quantity := range contents.Commander {
		check := deck.Contents.Commander[id]
		if check != 0 {
			deck.Contents.Commander[id] = check - quantity
		}

		if deck.Contents.Commander[id] == 0 {
			delete(deck.Contents.Commander, id)
		}
	}

	deck.MtgjsonApiMeta.ModifiedDate = util.CreateTimestampStr()

	// this is really in-efficient and should be changed
	err := ReplaceDeck(database, deck)
	if err != nil {
		return err
	}

	return nil
}
