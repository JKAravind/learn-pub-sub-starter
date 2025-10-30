package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const TransientQueue SimpleQueueType = 0
const DurableQueue SimpleQueueType = 1

type AckType int

const Ack AckType = 0
const NackRequeue AckType = 1
const NackDiscard AckType = 2

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

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {

	var buf bytes.Buffer
	Encoder := gob.NewEncoder(&buf)
	err := Encoder.Encode(val)

	if err != nil {
		fmt.Print(err)
		return err
	}
	ctx := context.Background()
	msg := amqp.Publishing{
		ContentType: "application/gob",
		Body:        buf.Bytes(),
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
	q, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, noWait, nil)
	if err != nil {
		ch.Close()
		return nil, amqp.Queue{}, fmt.Errorf("queue declare %s: %w", queueName, err)
	}

	if err := ch.QueueBind(queueName, key, exchange, noWait, nil); err != nil {
		ch.Close()
		return nil, amqp.Queue{}, fmt.Errorf("queue bind %s -> %s: %w", queueName, key, err)
	}

	return ch, q, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {

	QueueBindChannel, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("DeclareAndBind failed: %w", err)
	}
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
			ackType := handler(unMarsheledData)
			switch ackType {
			case Ack:
				message.Ack(false)
				fmt.Println("Ack is triggered Succes consumed")
			case NackRequeue:
				message.Nack(false, true)
				fmt.Println("NAck is triggered Succes Requeu")
			case NackDiscard:
				message.Nack(false, false)
				fmt.Println("NAck is triggered Succes Discarded")

			}

		}
	}()

	return nil
}
