package grpc_report

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/grpc/report/api"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/model/messages"
	"gitlab.ozon.dev/netrebinr/netrebin-roman/internal/repository"
	"google.golang.org/grpc"
)

type ReportServer struct {
	model    *messages.Model
	tgClient *messages.MessageSender
	api.UnimplementedReportCollectorServer
}

func NewServer(model *messages.Model, tgClient messages.MessageSender) error {
	lis, err := net.Listen("tcp", ":5000")
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}
	s := grpc.NewServer()
	api.RegisterReportCollectorServer(s, &ReportServer{
		model:    model,
		tgClient: &tgClient,
	})
	if err := s.Serve(lis); err != nil {
		return errors.Wrap(err, "failed to serve")
	}
	return nil
}

func (s *ReportServer) ReceiveReport(ctx context.Context, in *api.ReportBody) (*api.ReportAccept, error) {
	log.Printf("received report for userid %v", in.UserId)

	var reportByCategory []*repository.ReportByCategory
	for _, row := range in.Repcat {
		sum, _ := decimal.NewFromString(row.Sum)
		reportByCategory = append(reportByCategory, &repository.ReportByCategory{
			CategoryName: row.CategoryName,
			Sum:          sum,
		})
	}
	report := &repository.Report{
		ReportByCategory: reportByCategory,
		MinDate:          time.Unix(in.MinDate, 0),
	}

	message, err := s.model.ProceedCommandReport(ctx, in.UserId, report)
	if err != nil {
		return nil, err
	}
	err = (*s.tgClient).SendMessage(ctx, message, in.UserId)

	return &api.ReportAccept{Answer: "ok"}, err
}
