package main

import (
	"log"
	"github.com/readysetliqd/airdrop-discord-bot-go/bot"
	"github.com/readysetliqd/airdrop-discord-bot-go/config"
)

func main() {
	// load configs, secrets, and backup data into memory
	loadConfig()
	err := bot.Start()
	if err != nil {
		log.Fatal(err)
	}

	//TODO move this to setup
	//DEBUG
	// emojis, err := s.GuildEmojis("1227637875428950136")
	// if err != nil {
	// 	log.Println(err)
	// }
	// fmt.Println("emojis")
	// for _, emoji := range emojis {
	// 	fmt.Println(*emoji)
	// }
	//^DEBUG
}

// loadConfig loads the config.json file into the config struct to be used by
// any package importing the config package. It logs fatal on any errors
func loadConfig() {
	err := config.ReadConfig()
	if err != nil {
		log.Fatal("error reading configs |", err)
	}
}
