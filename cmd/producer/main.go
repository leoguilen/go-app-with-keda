package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type OrderStatus string

const (
	Pending OrderStatus = "Pending"
	Processing
	Done
)

type Order struct {
	Id         int         `json:"id"`
	CustomerId int         `json:"customerId"`
	Date       time.Time   `json:"date"`
	Amount     float64     `json:"amount"`
	Status     OrderStatus `json:"status"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func getPendingOrdersBatch(ctx context.Context, batchSize int) ([]Order, error) {
	conn, err := _db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.QueryContext(ctx, "select * from orders where status = 'Pending' limit $1", batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order

	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.Id, &order.CustomerId, &order.Date, &order.Amount, &order.Status); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func publishOrdersToProcess(ctx context.Context, orders []Order) error {
	ch, err := _broker.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	failOnError(ch.Confirm(false), "Failed to enable confirmation mode")

	q, err := ch.QueueDeclare(
		"pending_orders",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	for _, order := range orders {
		body, err := json.Marshal(order)
		failOnError(err, "Failed to marshal order")

		err = ch.PublishWithContext(ctx,
			"",
			q.Name,
			false,
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json",
				Body:         body,
			})
		failOnError(err, "Failed to publish a message")

		if confirmed := <-confirms; confirmed.Ack {
			log.Printf("Push confirmed [%d]!", confirmed.DeliveryTag)
		}
	}

	return nil
}

func updateOrdersStatus(ctx context.Context, orders []Order) error {
	conn, err := _db.Conn(ctx)
	failOnError(err, "Failed to get connection")
	defer conn.Close()

	var ordersIds []int
	for _, order := range orders {
		ordersIds = append(ordersIds, order.Id)
	}

	res, err := conn.ExecContext(ctx, "update orders set status = 'Processing' where id = any($1)", pq.Array(ordersIds))
	failOnError(err, "Failed to update orders status")

	rowsAffected, err := res.RowsAffected()
	failOnError(err, "Failed to get rows affected")

	log.Printf("rowsAffected: %d", rowsAffected)
	return nil
}

var (
	_db     *sql.DB
	_broker *amqp.Connection
)

func init() {
	log.SetPrefix("[Producer] - ")

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URI"))
	failOnError(err, "Failed to open database connection")

	broker, err := amqp.Dial(os.Getenv("RABBITMQ_URI"))
	failOnError(err, "Failed to open rabbitmq connection")

	_db = db
	_broker = broker

	log.Print("Application initialized")
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-ticker.C:
				orders, err := getPendingOrdersBatch(ctx, 100)
				if err != nil {
					log.Fatal(err)
					return
				}
				go publishOrdersToProcess(ctx, orders)
				go updateOrdersStatus(ctx, orders)
				log.Printf("Orders sent to process: %v", orders)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}(ctx)

	wg.Wait()

	if err := _db.Close(); err != nil {
		log.Panic(err)
	}
	if err := _broker.Close(); err != nil {
		log.Panic(err)
	}

	log.Println("Application stopped")
}
