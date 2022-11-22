package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	_ "github.com/lib/pq"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/cache/cache_lru"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/clients/tg"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/config"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/currency/cbrcurrency"
	grpc_report "gitlab.ozon.dev/netrebinr/netrebin-roman/internal/grpc/report/server"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/kafka/producers"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/model/messages"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/observability"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository/postgres_sql"
	"go.uber.org/zap"
)

var (
	develMode = flag.Bool("devel", false, "development mode")

	KafkaTopic  = "report-requests"
	BrokersList = []string{"kafka:9092"}
)

func main() {
	ctx, cancelFn := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	flag.Parse()

	logger := observability.InitLogger(*develMode)

	observability.InitTracing(logger)

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

	db, err := postgres_sql.OpenAndConnect("build/bot/environment.dev")
	if err != nil {
		logger.Fatal(err.Error())
	}
	spRepository := postgres_sql.New(db)

	currencyCache := cache_lru.NewLRUCache("currency", config.CurrencyCacheSize())
	reportCache := cache_lru.NewLRUCache("report", config.ReportCacheSize())

	cbrCurrency, err := cbrcurrency.NewCbrCurrencyStorage(ctx, &wg, logger)
	if err != nil {
		logger.Warn(
			"currency storage temporary failed",
			zap.Error(err),
		)
	}

	reportService, err := producer.New(producer.ProducerOptions{
		KafkaTopic:  KafkaTopic,
		BrokersList: BrokersList,
	})
	if err != nil {
		logger.Fatal(err.Error())
	}

	msgModel := messages.New(tgClient, spRepository, currencyCache, reportCache, cbrCurrency, reportService)

	go func() {
		err = grpc_report.NewServer(msgModel, tgClient)
		if err != nil {
			logger.Fatal(err.Error())
		}
	}()

	go tgClient.ListenUpdates(ctx, &wg, msgModel)

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan
	cancelFn()

	metricsSrv.Stop()

	wg.Wait()
}
