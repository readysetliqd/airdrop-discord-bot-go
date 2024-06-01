package main

import (
	"fmt"
	"log"
	"time"

	"github.com/readysetliqd/airdrop-discord-bot-go/bot"
	"github.com/readysetliqd/airdrop-discord-bot-go/config"
)

func main() {
	err := config.ReadConfig()

	if err != nil {
		fmt.Println(err)
		return
	}

	goBot := bot.Start()

	tickerDelay := 10 * time.Second
	ticker := time.NewTicker(tickerDelay)

	for range ticker.C {
		goBot.ChannelMessageSend(config.DefaultChannelID, "times up!")
		log.Println("timer fired to channel ", config.DefaultChannelID)
		ticker.Reset(tickerDelay)
	}

	<-make(chan struct{})
	return
}
