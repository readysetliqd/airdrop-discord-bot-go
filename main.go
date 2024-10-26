package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/readysetliqd/airdrop-discord-bot-go/bot"
	"github.com/readysetliqd/airdrop-discord-bot-go/config"
	"github.com/readysetliqd/airdrop-discord-bot-go/data"
)

var s *discordgo.Session
var protocols *data.Protocols
var unprocessedMessages *data.UnprocessedMessages

func main() {
	// load configs, secrets, and backup data into memory
	loadConfig()
	loadGoogleSecrets()
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

// loadGoogleSecrets is a helper function that loads the google secrets from
// the .env file into the os environment. It shuts down program on any errors
func loadGoogleSecrets() {
	err := godotenv.Load(data.GoogleSecretsEnvFileName)
	if err != nil {
		log.Fatal("error loading .env files |", err)
	}
}

// loadConfig loads the config.json file into the config struct to be used by
// any package importing the config package. It logs fatal on any errors
func loadConfig() {
	err := config.ReadConfig()
	if err != nil {
		log.Fatal("error reading configs |", err)
	}
}
