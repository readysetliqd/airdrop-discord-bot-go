package config

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	Token               string
	BotPrefix           string
	DefaultChannelID    string
	GuildID             string
	FundingRoundRoleID  string
	EarlyRoundRoleID    string
	BinanceRoundRoleID  string
	ParadigmRoundRoleID string
	CoinbaseRoundRoleID string
	BotOperatorRoleID   string
	TwitterEmoji        string
	TwitterEmojiName    string

	config *Config
)

type Config struct {
	Token               string `json:"token"`
	BotPrefix           string `json:"botPrefix"`
	DefaultChannelID    string `json:"channelID"`
	GuildID             string `json:"guildID"`
	FundingRoundRoleID  string `json:"fundingRoundRoleID"`
	EarlyRoundRoleID    string `json:"earlyRoundRoleID"`
	BinanceRoundRoleID  string `json:"binanceRoundRoleID"`
	ParadigmRoundRoleID string `json:"paradigmRoundRoleID"`
	CoinbaseRoundRoleID string `json:"coinbaseRoundRoleID"`
	BotOperatorRoleID   string `json:"botOperatorRoleID"`
	TwitterEmojiName    string `json:"twitterEmojiName"`
	TwitterEmojiID      string `json:"twitterEmojiID"`
}

// ReadConfig reads the config.json file and unmarshals it into the Config struct
func ReadConfig() error {
	// read config file in entirety
	fmt.Println("Reading config.json...")
	file, err := os.ReadFile("./config.json")
	if err != nil {
		return err
	}

	// unmarshal file into config struct
	fmt.Println("Unmarshalling config.json...")
	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Error unmarshalling config.json")
		return err
	}

	Token = config.Token
	BotPrefix = config.BotPrefix
	DefaultChannelID = config.DefaultChannelID
	GuildID = config.GuildID
	//TODO: change all these two fmt.Sprintf()
	FundingRoundRoleID = "<@&" + config.FundingRoundRoleID + ">"
	EarlyRoundRoleID = "<@&" + config.EarlyRoundRoleID + ">"
	BinanceRoundRoleID = "<@&" + config.BinanceRoundRoleID + ">"
	ParadigmRoundRoleID = "<@&" + config.ParadigmRoundRoleID + ">"
	CoinbaseRoundRoleID = "<@&" + config.CoinbaseRoundRoleID + ">"
	BotOperatorRoleID = config.BotOperatorRoleID
	TwitterEmoji = fmt.Sprintf("%s:%s", config.TwitterEmojiName, config.TwitterEmojiID)
	TwitterEmojiName = config.TwitterEmojiName
	return nil
}
