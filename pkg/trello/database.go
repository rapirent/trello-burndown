package trello

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
)

// Board contains data of a trello board.
type Board struct {
	ID              string `gorm:"primary_key"`
	Name            string
	DateStart       time.Time
	DateEnd         time.Time
	Cards           uint
	Points          float64
	CardsCompleted  uint
	PointsCompleted float64
	CardProgress    []CardProgress
}

// CardProgress represents the progress of a card.
type CardProgress struct {
	gorm.Model
	BoardID string
	Date    time.Time
	Points  float64
}

type BoardProgress struct {
	gorm.Model
	BoardID string
	Date    time.Time
	PointsCompleted  float64
}

// GetDatabase returns a sqlite3 database connection.
func GetDatabase() *gorm.DB {
	db, err := gorm.Open(viper.GetString("database.dialect"), viper.GetString("database.url"))
	if err != nil {
		log.Fatalln(err)
	}
	db.AutoMigrate(&Board{}, &CardProgress{}, &BoardProgress{})
	return db
}

func saveProgressToDatabase(board Board, pointsToday float64) {
	db := GetDatabase()
	defer db.Close()
	oldBoard := Board{}
	db.Where("id = ?", board.ID).First(&oldBoard)
	db.Model(oldBoard).Updates(&board)
	dateYst := time.Now().AddDate(0, 0, -1)
	yesterday := time.Date(dateYst.Year(), dateYst.Month(), dateYst.Day(), 0, 0, 0, 0, dateYst.Location())
	ytdBoardProgress := BoardProgress{}
	var pointsYesterday float64 = 0
	result := db.Where("date = ? AND board_id =?", yesterday, board.ID).First(&ytdBoardProgress)
	if result.Error == nil {
		pointsYesterday = ytdBoardProgress.PointsCompleted
	}
	dateNow := time.Now()
	today := time.Date(dateNow.Year(), dateNow.Month(), dateNow.Day(), 0, 0, 0, 0, dateNow.Location())
	oldCardProgress := CardProgress{}
	newCardProgress := CardProgress{
		Date:    today,
		Points:  pointsToday - pointsYesterday,
		BoardID: board.ID,
	}
	result = db.Where("date = ? AND board_id = ?", today, board.ID).First(&oldCardProgress)
	if result.Error != nil {
		db.Save(&newCardProgress)
	} else {
		db.Model(oldCardProgress).Updates(&newCardProgress)
	}

	oldBoardProgress := BoardProgress{}
	newBoardProgress := BoardProgress{
		Date: today,
		PointsCompleted: pointsToday,
		BoardID: board.ID,
	}
	result = db.Where("date = ? AND board_id = ?", today, board.ID).First(&oldBoardProgress)
	if result.Error != nil {
		db.Save(&newBoardProgress)
	} else {
		db.Model(oldBoardProgress).Updates(&oldBoardProgress)
	}
}
