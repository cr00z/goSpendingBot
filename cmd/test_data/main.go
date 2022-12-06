package main

import (
	"context"
	"database/sql"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/cr00z/goSpendingBot/internal/repository/postgres_sql"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
)

const (
	recordLimit          = 20_000
	usersLimit           = 100
	categoryPerUserLimit = 10
	historyDepthLimit    = 365
)

func main() {
	ctx := context.Background()

	dsn := "host=localhost port=5432 user=postgres password=qwerty sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("db connect failed: ", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("db connect failed: ", err)
	}

	ps := postgres_sql.New(db)

	for i := 1; i < recordLimit; i++ {
		if i%1000 == 0 {
			log.Printf("%d/%d records added", i, recordLimit)
		}
		userId := int64(893762098) + int64(rand.Intn(usersLimit)) // myId = 893762098
		amount := decimal.NewFromFloat(float64(rand.Intn(1000)) / 100)
		category := "category" + strconv.Itoa(rand.Intn(categoryPerUserLimit))
		time := time.Now().AddDate(0, 0, -rand.Intn(historyDepthLimit))
		err := ps.CreateSpending(ctx, userId, category, amount, time)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Println("done")
}
