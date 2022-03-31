package trello

import (
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/adlio/trello"
	"github.com/jasonlvhit/gocron"
	"github.com/spf13/viper"
)

type cardResult struct {
	ID			string
	Error       error
	Date        string
	Complete    bool
	Points      float64
	TotalPoints float64
	TrelloError bool
}

var client *trello.Client

// Start starts watching boards that are active. Refreshes according
// to the refresh rate set in the configuration.
func Start() {
	client = trello.NewClient(
		viper.GetString("trello.apiKey"),
		viper.GetString("trello.userToken"),
	)
	go runBoards()
	ch := gocron.Start()
	refreshRate := uint64(viper.GetInt64("trello.refreshRate"))
	gocron.Every(refreshRate).Minutes().Do(runBoards)
	<-ch
}

func runBoards() {
	db := GetDatabase()
	defer db.Close()
	boards := []Board{}
	yesterday := time.Now().Add(-24 * time.Hour)
	db.Select("id").Where("date_start < ? AND date_end > ?", yesterday, yesterday).Find(&boards)
	for _, board := range boards {
		go Run(board.ID)
	}
}

// Run fetches and saves the points of a given board.
func Run(boardID string) {
	log.Printf("Checking board ID %s", boardID)
	board, err := client.GetBoard(boardID, trello.Defaults())
	if err != nil {
		log.Printf("Couldn't fetch board: %s", err)
		return
	}
	log.Printf("Board name: %s", board.Name)
	lastListID, err := getLastList(board)
	if err != nil {
		log.Printf("Couldn't fetch last list: %s", err)
	}
	resultChannel := make(chan *cardResult)
	cards, err := board.GetCards(trello.Defaults())
	if err != nil {
		log.Printf("Couldn't fetch cards: %s", err)
	}
	for _, card := range cards {
		go determineCardComplete(card, lastListID, resultChannel)
	}
	boardEntity := Board{
		ID:   boardID,
		Name: board.Name,
	}
	for i := 0; i < len(cards); i++ {
		response := <-resultChannel
		if response.Error != nil {
			log.Fatalln(response.Error)
		}
		if response.TotalPoints == 0 {
			continue
		}
		if response.Complete {
			boardEntity.CardsCompleted++
		}
		boardEntity.Points += response.TotalPoints
		boardEntity.PointsCompleted += response.Points
		boardEntity.Cards++
	}
	log.Printf("Cards progress: %d/%d", boardEntity.CardsCompleted, boardEntity.Cards)
	log.Printf("Total points: %f/%f", boardEntity.PointsCompleted, boardEntity.Points)
	saveProgressToDatabase(boardEntity, boardEntity.PointsCompleted)
}

func getLastList(board *trello.Board) (string, error) {
	var highestPos float32
	var listID string
	lists, err := board.GetLists(trello.Defaults())
	if err != nil {
		return "", err
	}
	for _, list := range lists {
		if list.Pos > highestPos {
			listID = list.ID
		}
	}
	return listID, nil
}

func determineCardComplete(card *trello.Card, listID string, res chan *cardResult) {
	points, totalPoints := getPoints(card)
	actions, err := card.GetActions(trello.Defaults())
	if err != nil {
		res <- &cardResult{
			Error: err,
		}
		return
	}
	if card.IDList != listID {
		date :=time.Now()
		res <- &cardResult{
			ID: 			card.ID,
			Complete:		false,
			Date:     		date.Format("2006-01-02"),
			TotalPoints: 	totalPoints,
			Points:   		points,
		}
		return
	}
	date := card.DateLastActivity
	for _, action := range actions {
		if action.Data.ListAfter != nil && action.Data.ListBefore != nil &&
			action.Data.ListAfter.ID != action.Data.ListBefore.ID && action.Data.ListAfter.ID == listID {
			date = &action.Date
			break
		}
	}
	res <- &cardResult{
		ID: card.ID,
		Complete: true,
		Date:     date.Format("2006-01-02"),
		TotalPoints: totalPoints,
		Points:   points,
	}
}

func getPoints(card *trello.Card) (float64,float64) {
	r := regexp.MustCompile(`\(([0-9]*\.[0-9]+|[0-9]+)\)`)
	matches := r.FindStringSubmatch(card.Name)
	if len(matches) != 2 {
		return 0, 0
	}
	weight, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		log.Fatalln(err)
	}
	var points float64 = float64(card.Badges.CheckItemsChecked)
	var TotalPoints float64 = float64(card.Badges.CheckItems)
	if (points == 0 || TotalPoints == 0) && weight != 0 {
		return 0, weight
	}
	return points*weight, TotalPoints*weight
}
