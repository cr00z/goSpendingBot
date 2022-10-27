package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/clients/tg"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/config"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/currency/cbrcurrency"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/model/messages"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/observability"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository/postgres_sql"
	"go.uber.org/zap"
)

var (
	develMode = flag.Bool("devel", false, "development mode")
)

func main() {
	flag.Parse()
	logger := observability.InitLogger(*develMode)

	ctx, cancelFn := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	config, err := config.New()
	if err != nil {
		logger.Fatal("config init failed: ", zap.Error(err))
	}

	metricsSrv := observability.NewMetricsServer(logger)
	metricsSrv.Start()

	tgClient, err := tg.New(config, logger)
	if err != nil {
		logger.Fatal("tg client init failed: ", zap.Error(err))
	}

	if err := godotenv.Load("build/bot/environment.dev"); err != nil {
		logger.Fatal("error loading env variables: ", zap.Error(err))
	}
	dsn := fmt.Sprintf("host=%s port=5432 user=%s password=%s sslmode=%s",
		os.Getenv("POSTGRES_HOST"),
		// "localhost",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_SSLMODE"),
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Fatal("db connect failed: ", zap.Error(err))
	}
	if err = db.Ping(); err != nil {
		logger.Fatal("db connect failed: ", zap.Error(err))
	}

	spRepository := postgres_sql.New(db)

	cbrCurrency, err := cbrcurrency.NewCbrCurrencyStorage(ctx, &wg, logger)
	if err != nil {
		logger.Warn(
			"currency storage temporary failed",
			zap.Error(err),
		)
	}

	msgModel := messages.New(tgClient, spRepository, cbrCurrency)

	go tgClient.ListenUpdates(ctx, &wg, msgModel)

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan
	cancelFn()

	metricsSrv.Stop()

	wg.Wait()
}
