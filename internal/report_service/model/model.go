package model

import "gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository"

type ReportServiceModel struct {
	Store repository.Storager
}

func New(store repository.Storager) *ReportServiceModel {
	return &ReportServiceModel{
		Store: store,
	}
}
