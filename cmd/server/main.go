package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	const connString = "amqp://guest:guest@localhost:5672/"
	amqpConn, err := amqp.Dial(connString)
	if err != nil {
		panic(err)
	}
	defer amqpConn.Close()

	fmt.Println("Connected to RabbitMQ")

	newChannel, err := amqpConn.Channel()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("New Channel created")
	exchange := routing.ExchangePerilDirect
	routingKey := "game_logs.*"
	queueName := "game_logs"

	queueCreationChannel, createdQueue, err := pubsub.DeclareAndBind(amqpConn, exchange, queueName, routingKey, pubsub.DurableQueue)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print(createdQueue, queueCreationChannel)

	gamelogic.PrintServerHelp()

	for {
		words := gamelogic.GetInput()
		if words[0] == "pause" {
			fmt.Print("Pausing")
			exchangeString := routing.ExchangePerilDirect
			routingKey := routing.PauseKey

			jsonBodyNeededToBeSent := routing.PlayingState{
				IsPaused: true,
			}
			didJSONPublishedError := pubsub.PublishJSON(newChannel, exchangeString, routingKey, jsonBodyNeededToBeSent)
			if didJSONPublishedError != nil {
				fmt.Println(didJSONPublishedError)
			}
			fmt.Println(didJSONPublishedError)
		} else if words[0] == "resume" {
			fmt.Println("Resuming")
			exchangeString := routing.ExchangePerilDirect
			routingKey := routing.PauseKey

			jsonBodyNeededToBeSent := routing.PlayingState{
				IsPaused: false,
			}
			didJSONPublishedError := pubsub.PublishJSON(newChannel, exchangeString, routingKey, jsonBodyNeededToBeSent)
			if didJSONPublishedError != nil {
				fmt.Println(didJSONPublishedError)
			}
			fmt.Println(didJSONPublishedError)

		} else if words[0] == "quit" {
			fmt.Println("Exiting")
			break
		} else {
			fmt.Println("Give a Correct dCmmand")

		}
	}

	newChannel.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	sig := <-sigCh
	fmt.Println("Received signal:", sig)
	if err := amqpConn.Close(); err != nil {
		log.Printf("Error closing connection: %v", err)
	}
	fmt.Println("Shutting down gracefully...")
}
