package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	amqpConn, err := amqp.DialConfig("amqp://guest:guest@localhost:5672/", amqp.Config{
		Heartbeat: 60 * time.Second,
		Locale:    "en_US",
		Dial:      amqp.DefaultDial(10 * time.Second),
	})
	if err != nil {
		panic(err)
	}
	defer amqpConn.Close()

	fmt.Println("Connected to RabbitMQ")

	newChannel, err := amqpConn.Channel()
	WatchChannel("publisher", newChannel)

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
	armyMovesHandlerFunction := handerArmyMove(UserCreatedGameState, newChannel)
	pubsub.SubscribeJSON(amqpConn, armyMovesExchange, armyMovesQueueName, armyMovesRoutingKey, pubsub.TransientQueue, armyMovesHandlerFunction)

	//This is for warMoves Queue
	warInforExchange := routing.ExchangePerilTopic
	warInfoRoutingKey := "war.*"
	warInfoQueueName := "war"
	warInfoHandler := handlerGetMoveMessages(UserCreatedGameState, newChannel)
	pubsub.SubscribeJSON(amqpConn, warInforExchange, warInfoQueueName, warInfoRoutingKey, pubsub.DurableQueue, warInfoHandler)

	done := make(chan struct{})

	go func() {
	loop:
		for {
			words := gamelogic.GetInput()
			switch words[0] {
			case "spawn":
				if err := UserCreatedGameState.CommandSpawn(words); err != nil {
					fmt.Println(err)
				}
			case "move":
				armyMove, err := UserCreatedGameState.CommandMove(words)
				if err != nil {
					fmt.Println(err)
					continue
				}
				if err := pubsub.PublishJSON(newChannel, armyMovesExchange, armyMovesQueueName, armyMove); err != nil {
					fmt.Println(err)
				} else {
					fmt.Println("Moved Success")
				}
			case "status":
				UserCreatedGameState.CommandStatus()
			case "help":
				gamelogic.PrintClientHelp()
			case "quit":
				gamelogic.PrintQuit()
				break loop
			default:
				fmt.Println("Unknown command. Type 'help' for options.")
			}
		}
		close(done)
	}()

	// Keep the main goroutine alive for heartbeats & subscribers.
	<-done

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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Println("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}

}

func handerArmyMove(gameState *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(armyMove gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Println("> ")
		moveOutCome := gameState.HandleMove(armyMove)
		switch moveOutCome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			routingKey := fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, gameState.Player.Username)
			messageWarRecognition := gamelogic.RecognitionOfWar{
				Attacker: gameState.Player,
				Defender: armyMove.Player,
			}
			err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, routingKey, messageWarRecognition)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerGetMoveMessages(gamestate *gamelogic.GameState, channel *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(warDetail gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Println(">  ")
		WarOutcome, winner, loser := gamestate.HandleWar(warDetail)
		switch WarOutcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			LogMessage := fmt.Sprintf("%s won a war against %s", winner, loser)
			err := publishGameLog(LogMessage, gamestate.GetUsername(), channel)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			LogMessage := fmt.Sprintf("%s won a war against %s", winner, loser)
			err := publishGameLog(LogMessage, gamestate.GetUsername(), channel)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			LogMessage := fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			err := publishGameLog(LogMessage, gamestate.GetUsername(), channel)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		default:
			return pubsub.Ack
		}

	}
}

func WatchChannel(name string, ch *amqp.Channel) {
	go func() {
		for err := range ch.NotifyClose(make(chan *amqp.Error)) {
			if err != nil {
				log.Printf("❌ [%s] Channel closed: %v", name, err)
			} else {
				log.Printf("⚠️ [%s] Channel closed cleanly", name)
			}
		}
	}()
}

func publishGameLog(LogMessage string, UserName string, channel *amqp.Channel) error {
	exchange := routing.ExchangePerilTopic
	routingKey := fmt.Sprintf("%s.%s", routing.GameLogSlug, UserName)
	gameLogStruct := routing.GameLog{
		CurrentTime: time.Now(),
		Username:    UserName,
		Message:     LogMessage,
	}
	err := pubsub.PublishGob(channel, exchange, routingKey, gameLogStruct)
	if err != nil {
		return err
	}
	return nil

}
