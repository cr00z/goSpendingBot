package model

import "github.com/cr00z/goSpendingBot/internal/repository"

type ReportServiceModel struct {
	Store repository.Storager
}

func New(store repository.Storager) *ReportServiceModel {
	return &ReportServiceModel{
		Store: store,
	}
}
