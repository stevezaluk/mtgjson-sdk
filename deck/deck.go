package deck

import (
	"errors"
	cardModel "github.com/stevezaluk/mtgjson-models/card"
	"github.com/stevezaluk/mtgjson-sdk/card"
	"github.com/stevezaluk/mtgjson-sdk/context"
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
func ReplaceDeck(deck *deckModel.Deck) error {
	var database = context.GetDatabase()

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
func DeleteDeck(code string) error {
	var database = context.GetDatabase()

	query := bson.M{"code": code}
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
GetDeck Fetch a deck from the MongoDB database using the code passed in the parameter. Returns
ErrNoDeck if the deck does not exist or cannot be located
*/
func GetDeck(code string) (*deckModel.Deck, error) {
	var result *deckModel.Deck

	var database = context.GetDatabase()

	query := bson.M{"code": code}
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
func IndexDecks(limit int64) ([]*deckModel.Deck, error) {
	var result []*deckModel.Deck

	var database = context.GetDatabase()

	err := database.Index("deck", limit, &result)
	if !err {
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
	if !errors.Is(err, sdkErrors.ErrNoDeck) {
		return sdkErrors.ErrDeckAlreadyExists
	}

	var database = context.GetDatabase()

	database.Insert("deck", &deck)

	return nil
}

/*
GetBoardContents Return a slice of CardSet pointers representing a deck boards content. If the requested board
does not exist, it will return ErrBoardNotExist
*/
func GetBoardContents(contentIds *deckModel.DeckContentIds, board string) ([]*cardModel.CardSet, error) {
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

	return card.GetCards(boardIds)
}

/*
GetDeckContents Update the 'contents' field of the deck passed in the parameter. This accepts a
pointer and updates this in place to avoid having to copy large amounts of data
*/
func GetDeckContents(deck *deckModel.Deck) error {
	if deck.ContentIds == nil {
		return sdkErrors.ErrDeckMissingId
	}

	mainBoardContents, _ := GetBoardContents(deck.ContentIds, BoardMainboard)
	sideBoardContents, _ := GetBoardContents(deck.ContentIds, BoardSideboard)
	commanderContents, _ := GetBoardContents(deck.ContentIds, BoardCommander)

	contents := &deckModel.DeckContents{
		MainBoard: mainBoardContents,
		SideBoard: sideBoardContents,
		Commander: commanderContents,
	}

	deck.Contents = contents

	return nil
}

/*
AllCardIds Helper function to combine all card id's in a deck into a a single slice of strings
*/
func AllCardIds(deck *deckModel.Deck) ([]string, error) {
	var ret []string

	if deck.ContentIds == nil {
		return ret, sdkErrors.ErrDeckMissingId
	}

	ret = append(ret, deck.ContentIds.MainBoard...)
	ret = append(ret, deck.ContentIds.SideBoard...)
	ret = append(ret, deck.ContentIds.Commander...)

	return ret, nil
}

/*
AddCards Update the content ids in the deck model passed with new cards. Does not make database calls,
use ReplaceDeck to update the deck with these values
*/
func AddCards(deck *deckModel.Deck, newCards *deckModel.DeckContentIds) error {
	if deck.ContentIds == nil {
		return sdkErrors.ErrDeckMissingId
	}

	deck.ContentIds.MainBoard = append(deck.ContentIds.MainBoard, newCards.MainBoard...)
	deck.ContentIds.SideBoard = append(deck.ContentIds.SideBoard, newCards.SideBoard...)
	deck.ContentIds.Commander = append(deck.ContentIds.Commander, newCards.Commander...)

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
RemoveCards Remove cards from the content ids in the deck model passed. Does not make database calls, use ReplaceDeck to update the deck with these values
*/
func RemoveCards(deck *deckModel.Deck, removeCards *deckModel.DeckContentIds) error {
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

	return nil
}
