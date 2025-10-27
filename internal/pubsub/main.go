package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const TransientQueue SimpleQueueType = 0
const DurableQueue SimpleQueueType = 1

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {

	jsonValSlice, err := json.Marshal(val)
	if err != nil {
		fmt.Print(err)
		return err
	}
	ctx := context.Background()
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonValSlice,
	}
	err = ch.PublishWithContext(ctx, exchange, key, false, false, msg)
	if err != nil {
		fmt.Print(err)
		return err
	}
	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	durable := false
	autoDelete := false
	exclusive := false
	noWait := false

	if queueType == DurableQueue {
		durable = true
	} else {
		autoDelete = true
		exclusive = true
	}
	newQueue, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, noWait, nil)
	if err != nil {
		fmt.Print(err)
	}
	ch.QueueBind(queueName, key, exchange, noWait, nil)
	return ch, newQueue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T),
) error {

	QueueBindChannel, _, _ := DeclareAndBind(conn, exchange, queueName, key, queueType)

	QueueMessagesChannel, err := QueueBindChannel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		fmt.Print(err)
	}

	go func() {
		for message := range QueueMessagesChannel {
			var unMarsheledData T
			err := json.Unmarshal(message.Body, &unMarsheledData)
			if err != nil {
				fmt.Print(err)
				continue
			}
			handler(unMarsheledData)
			err = message.Ack(false)
			if err != nil {
				fmt.Print(err)
			}

		}
	}()

	return nil
}
