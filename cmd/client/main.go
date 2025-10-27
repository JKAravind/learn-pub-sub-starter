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

	userName, _ := gamelogic.ClientWelcome()
	fmt.Println("User Name is:", userName)

	exchange := routing.ExchangePerilDirect
	routingKey := routing.PauseKey
	queueName := fmt.Sprintf("%s.%s", routingKey, userName)

	queueCreationChannel, createdQueue, err := pubsub.DeclareAndBind(amqpConn, exchange, queueName, routingKey, pubsub.TransientQueue)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print(createdQueue)
	UserCreatedGameState := gamelogic.NewGameState(userName)
	// This is For Subscribing the pause Queue
	handler := handlerPause(UserCreatedGameState)
	pubsub.SubscribeJSON(amqpConn, exchange, queueName, routingKey, pubsub.TransientQueue, handler)

	//This is for subscribing the Moves Queue
	armyMovesExchange := routing.ExchangePerilTopic
	armyMovesRoutingKey := "army_moves.*"
	armyMovesQueueName := fmt.Sprintf("army_moves.%s", userName)
	armyMovesHandlerFunction := handerArmyMove(UserCreatedGameState)
	pubsub.SubscribeJSON(amqpConn, armyMovesExchange, armyMovesQueueName, armyMovesRoutingKey, pubsub.TransientQueue, armyMovesHandlerFunction)

loop:
	for {
		words := gamelogic.GetInput()
		switch words[0] {
		case "spawn":
			err := UserCreatedGameState.CommandSpawn(words)
			if err != nil {
				fmt.Print(err)
			}
		case "move":
			armyMove, err := UserCreatedGameState.CommandMove(words)
			if err != nil {
				fmt.Println(err)
			}
			jsonBodyNeededToBeSentMove := armyMove
			err = pubsub.PublishJSON(newChannel, armyMovesExchange, armyMovesQueueName, jsonBodyNeededToBeSentMove)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Print("Moved Success")
		case "status":
			UserCreatedGameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "quit":
			gamelogic.PrintQuit()
			break loop // exits the main loop cleanly
		default:
			fmt.Println("Unknown command. Type 'help' for options.")
		}
	}

	queueCreationChannel.Close()
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Println("> ")
		gs.HandlePause(ps)
	}

}

func handerArmyMove(gameState *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(armyMove gamelogic.ArmyMove) {
		defer fmt.Println("> ")
		gameState.HandleMove(armyMove)
	}
}
