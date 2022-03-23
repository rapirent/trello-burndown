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

type CardRecord struct {
	ID         string `gorm:"primary_key"`
	LastPoints float64
}

// GetDatabase returns a sqlite3 database connection.
func GetDatabase() *gorm.DB {
	db, err := gorm.Open(viper.GetString("database.dialect"), viper.GetString("database.url"))
	if err != nil {
		log.Fatalln(err)
	}
	db.AutoMigrate(&Board{}, &CardProgress{}, &CardRecord{})
	return db
}

func getCardLastPointsFromDatabase(ID string) float64 {
	db := GetDatabase()
	defer db.Close()
	oldCard := CardRecord{}
	db.Where("id = ?", ID).First(&oldCard)

	return oldCard.LastPoints
}

func saveProgressToDatabase(board Board, pointsToday float64) {
	db := GetDatabase()
	defer db.Close()
	oldBoard := Board{}
	db.Where("id = ?", board.ID).First(&oldBoard)
	db.Model(oldBoard).Updates(&board)
	dateNow := time.Now()
	date := time.Date(dateNow.Year(), dateNow.Month(), dateNow.Day(), 0, 0, 0, 0, dateNow.Location())
	oldCardProgress := CardProgress{}
	newCardProgress := CardProgress{
		Date:    date,
		Points:  pointsToday,
		BoardID: board.ID,
	}
	result := db.Where("date = ?", date).First(&oldCardProgress)
	if result.Error != nil {
		db.Save(&newCardProgress)
		return
	}
	db.Model(oldCardProgress).Updates(&newCardProgress)
}

func saveCardToDatabase(card CardRecord) {
	db := GetDatabase()
	defer db.Close()
	oldCard := CardRecord{}
	results := db.Where("id = ?", card.ID).First(&oldCard)
	if results.Error != nil {
		db.Save(&card)
		return
	}
	db.Model(oldCard).Updates(&card)
}
