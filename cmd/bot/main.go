package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/clients/tg"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/config"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/currency/cbrcurrency"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/model/messages"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository/memory"
)

func main() {
	ctx, cancelFn := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	config, err := config.New()
	if err != nil {
		log.Fatal("config init failed: ", err)
	}

	tgClient, err := tg.New(config)
	if err != nil {
		log.Fatal("tg client init failed: ", err)
	}

	spRepository := memory.NewMemoryStorage()

	cbrCurrency, err := cbrcurrency.NewCbrCurrencyStorage(ctx, &wg)
	if err != nil {
		log.Println("currency storage temporary failed: ", err)
	}

	msgModel := messages.New(tgClient, spRepository, cbrCurrency)

	go tgClient.ListenUpdates(ctx, &wg, msgModel)

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan
	cancelFn()
	wg.Wait()
}
