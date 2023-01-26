package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/trunov/gophermart/internal/app/postgres"
	"github.com/trunov/gophermart/internal/app/util"
	"golang.org/x/sync/errgroup"
)

type MyAPIError struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type OrderInfo struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

type Workerpool struct {
	dbStorage            postgres.DBStorager
	accrualSystemAddress string
}

type Job interface {
	Run(ctx context.Context) error
}

type UpdateOrderJob struct {
	dbStorage   postgres.DBStorager
	orderNumber string
	client      *resty.Client
}

func NewWorkerpool(dbStorage *postgres.DBStorager, accrualSystemAddress string) *Workerpool {
	return &Workerpool{dbStorage: *dbStorage, accrualSystemAddress: accrualSystemAddress}
}

func (j *UpdateOrderJob) Run(ctx context.Context) error {
	fmt.Printf("job %s has started\n", j.orderNumber)

	var responseErr MyAPIError
	var order OrderInfo

	_, err := j.client.R().
		SetHeader("Accept", "application/json").
		SetError(&responseErr).
		SetResult(&order).
		Get("/api/orders/" + j.orderNumber)
	if err != nil {
		return err
	}

	status, err := util.FindKeyByValue(order.Status)
	if err != nil {
		return err
	}

	err = j.dbStorage.UpdateOrder(ctx, order.Order, status, order.Accrual)
	if err != nil {
		return err
	}
	return nil
}

func (w *Workerpool) runPool(ctx context.Context, jobs chan Job) error {
	gr, ctx := errgroup.WithContext(ctx)
	for i := 0; i < runtime.GOMAXPROCS(runtime.NumCPU()-1); i++ {
		gr.Go(func() error {
			for {
				select {
				case job := <-jobs:
					err := job.Run(ctx)
					if err != nil {
						return err
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		})
	}
	return gr.Wait()
}

func (w *Workerpool) Start(ctx context.Context, inputCh chan string) {
	jobs := make(chan Job)

	client := resty.New().SetBaseURL(w.accrualSystemAddress)

	go func() {
		for inputCh != nil {
			v, ok := <-inputCh
			if !ok {
				inputCh = nil
				continue
			}

			jobs <- &UpdateOrderJob{
				dbStorage:   w.dbStorage,
				orderNumber: v,
				client:      client,
			}
		}
	}()

	defer func() {
		close(jobs)
	}()

	err := w.runPool(ctx, jobs)
	if err != nil {
		log.Println(err)
	}
}
