package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/trunov/gophermart/internal/app/postgres"
	"golang.org/x/sync/errgroup"
)

type MyAPIError struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type OrderInfo struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual string `json:"accrual"`
}

type Workerpool struct {
	dbStorage            postgres.DBStorager
	accrualSystemAddress string
}

type Job interface {
	Run(ctx context.Context) error
}

type UpdateOrderJob struct {
	dbStorage            postgres.DBStorager
	accrualSystemAddress string
	orderNumber          string
	client               *resty.Client
}

func NewWorkerpool(dbStorage *postgres.DBStorager, accrualSystemAddress string) *Workerpool {
	return &Workerpool{dbStorage: *dbStorage, accrualSystemAddress: accrualSystemAddress}
}

func (j *UpdateOrderJob) Run(ctx context.Context) error {
	fmt.Printf("job %s has started\n", j.orderNumber)

	// go to accrual service /api/orders/{number}
	var responseErr MyAPIError
	var order OrderInfo

	_, err := j.client.R().
		SetError(&responseErr).
		SetResult(&order).
		Get(j.accrualSystemAddress + "/" + j.orderNumber)

	if err != nil {
		fmt.Println(responseErr)
		panic(err)
	}

	fmt.Println(order)

	// err := j.dbStorage.UpdateOrder(ctx, j.orderNumber, j.orderStatus)
	// if err != nil {
	// 	return err
	// }
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

	// make resty client, pass it to UpdateOrderJob
	client := resty.New()

	go func() {
		for inputCh != nil {
			v, ok := <-inputCh
			if !ok {
				inputCh = nil
				continue
			}
			// send request to get to know status
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
