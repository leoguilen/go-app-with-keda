package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
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

func updateOrderStatus(ctx context.Context, orderId int) error {
	conn, err := _db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %s", err.Error())
	}
	defer conn.Close()

	res, err := conn.ExecContext(ctx, "update orders set status = 'Done' where id = $1", orderId)
	if err != nil {
		return fmt.Errorf("failed to update order status: %s", err.Error())
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %s", err.Error())
	}

	log.Printf("rowsAffected: %d", rowsAffected)
	return nil
}

func processMessage(ctx context.Context, msg amqp.Delivery) {
	var order *Order
	if err := json.Unmarshal(msg.Body, &order); err != nil {
		log.Printf("failed to unmarshal message body: %s", msg.Body)
		msg.Nack(false, false)
		return
	}

	if err := updateOrderStatus(ctx, order.Id); err != nil {
		log.Printf("failed to update order status: %s", err.Error())
		msg.Nack(false, false)
		return
	}

	log.Printf("Processed order: %v", order)
	msg.Ack(true)
}

var (
	_db     *sql.DB
	_broker *amqp.Connection
)

func init() {
	log.SetPrefix("[Consumer] - ")

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

	ch, err := _broker.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"pending_orders",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	failOnError(ch.Qos(1, 0, false), "Failed to set QoS")

	msgs, err := ch.ConsumeWithContext(ctx, q.Name, "", false, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	go func(ctx context.Context) {
		for msg := range msgs {
			go processMessage(ctx, msg)
		}
	}(ctx)

	<-ctx.Done()

	if err := _db.Close(); err != nil {
		log.Panic(err)
	}
	if err := _broker.Close(); err != nil {
		log.Panic(err)
	}

	log.Println("Application stopped")
}
