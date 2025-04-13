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

/*
GetBoardContents Return a slice of CardSet pointers representing a deck boards content. If the requested board
does not exist, it will return ErrBoardNotExist
*/
func GetBoardContents(database *server.Database, contentIds *deckModel.DeckContentIds, board string) ([]*cardModel.CardSet, error) {
	var boardIds []string

	if board == BoardMainboard {
		boardIds = contentIds.MainBoard
	} else if board == BoardSideboard {
		boardIds = contentIds.SideBoard
	} else if board == BoardCommander {
		boardIds = contentIds.Commander
	} else {
		return nil, sdkErrors.ErrBoardNotExist
	}

	return card.GetCards(database, boardIds)
}

/*
GetDeckContents Update the 'contents' field of the deck passed in the parameter. This accepts a
pointer and updates this in place to avoid having to copy large amounts of data
*/
func GetDeckContents(database *server.Database, deck *deckModel.Deck) error {
	if deck.ContentIds == nil {
		return sdkErrors.ErrDeckMissingId
	}

	mainBoardContents, _ := GetBoardContents(database, deck.ContentIds, BoardMainboard)
	sideBoardContents, _ := GetBoardContents(database, deck.ContentIds, BoardSideboard)
	commanderContents, _ := GetBoardContents(database, deck.ContentIds, BoardCommander)

	contents := &deckModel.DeckContents{
		MainBoard: mainBoardContents,
		SideBoard: sideBoardContents,
		Commander: commanderContents,
	}

	deck.Contents = contents

	return nil
}

/*
AllCardIds Helper function to combine all card id's in a deck into a single slice of strings
*/
func AllCardIds(contents *deckModel.DeckContentIds) ([]string, error) {
	var ret []string

	if contents == nil {
		return ret, sdkErrors.ErrDeckMissingId
	}

	ret = append(ret, contents.MainBoard...)
	ret = append(ret, contents.SideBoard...)
	ret = append(ret, contents.Commander...)

	return ret, nil
}

/*
AddCards Update the content ids in the deck model passed with new cards. This should
probably validate cards in the future
*/
func AddCards(database *server.Database, deck *deckModel.Deck, newCards *deckModel.DeckContentIds) error {
	if deck.ContentIds == nil {
		return sdkErrors.ErrDeckMissingId
	}

	deck.ContentIds.MainBoard = append(deck.ContentIds.MainBoard, newCards.MainBoard...)
	deck.ContentIds.SideBoard = append(deck.ContentIds.SideBoard, newCards.SideBoard...)
	deck.ContentIds.Commander = append(deck.ContentIds.Commander, newCards.Commander...)

	deck.MtgjsonApiMeta.ModifiedDate = util.CreateTimestampStr() // need better error checking here

	err := ReplaceDeck(database, deck)
	if err != nil {
		return err
	}

	return nil
}

func RemoveCardsFromBoard(deck *deckModel.Deck, cards []string, board string) error {
	if deck.ContentIds == nil {
		return sdkErrors.ErrDeckMissingId
	}

	var sourceBoard *[]string
	if board == BoardMainboard {
		sourceBoard = &deck.ContentIds.MainBoard
	} else if board == BoardSideboard {
		sourceBoard = &deck.ContentIds.SideBoard
	} else if board == BoardCommander {
		sourceBoard = &deck.ContentIds.Commander
	} else {
		return sdkErrors.ErrBoardNotExist
	}

	for _, uuid := range cards {
		for index, value := range *sourceBoard {
			if value == uuid {
				*sourceBoard = slices.Delete(*sourceBoard, index, index+1)
			}
		}
	}

	return nil
}

/*
RemoveCards Remove cards from the content ids in the deck model passed.
*/
func RemoveCards(database *server.Database, deck *deckModel.Deck, removeCards *deckModel.DeckContentIds) error {
	if deck.ContentIds == nil {
		return sdkErrors.ErrDeckMissingId
	}

	err := RemoveCardsFromBoard(deck, removeCards.MainBoard, BoardMainboard)
	if err != nil {
		return err
	}

	err = RemoveCardsFromBoard(deck, removeCards.SideBoard, BoardSideboard)
	if err != nil {
		return err
	}

	err = RemoveCardsFromBoard(deck, removeCards.Commander, BoardCommander)
	if err != nil {
		return err
	}

	err = ReplaceDeck(database, deck)
	if err != nil {
		return err
	}

	return nil
}
