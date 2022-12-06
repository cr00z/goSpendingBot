package grpc_report

import (
	"context"
	"log"
	"time"

	"github.com/cr00z/goSpendingBot/internal/grpc/report/api"
	"github.com/cr00z/goSpendingBot/internal/repository"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var connStr = "gospend-bot:5000"

func SendReport(userID int64, period string, report *repository.Report) error {
	conn, err := grpc.Dial(connStr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return errors.Wrap(err, "did not connect")
	}
	defer conn.Close()
	c := api.NewReportCollectorClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var repcat []*api.ReportBody_ReportByCategory
	for _, row := range report.ReportByCategory {
		repcat = append(repcat, &api.ReportBody_ReportByCategory{
			CategoryName: row.CategoryName,
			Sum:          row.Sum.String(),
		})
	}
	body := api.ReportBody{
		UserId:  userID,
		Period:  period,
		Repcat:  repcat,
		MinDate: report.MinDate.Unix(),
	}
	r, err := c.ReceiveReport(ctx, &body)
	if err != nil {
		return errors.Wrap(err, "could not answer")
	}
	log.Printf("Answer: %s", r.GetAnswer())
	return nil
}
