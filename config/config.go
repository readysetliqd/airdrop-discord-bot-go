package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
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
)

// ReadConfig reads the environment variables and assigns them into global exported variables
func ReadConfig() error {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("loading .env | ", err)
	}
	Token = os.Getenv("TOKEN")
	BotPrefix = os.Getenv("BOTPREFIX")
	DefaultChannelID = os.Getenv("CHANNELID")
	GuildID = os.Getenv("GUILDID")
	FundingRoundRoleID = fmt.Sprintf("<@&%s>", os.Getenv("FUNDINGROUNDROLEID"))
	EarlyRoundRoleID = fmt.Sprintf("<@&%s>", os.Getenv("EARLYROUNDROLEID"))
	BinanceRoundRoleID = fmt.Sprintf("<@&%s>", os.Getenv("BINANCEROUNDROLEID"))
	ParadigmRoundRoleID = fmt.Sprintf("<@&%s>", os.Getenv("PARADIGMROUNDROLEID"))
	CoinbaseRoundRoleID = fmt.Sprintf("<@&%s>", os.Getenv("COINBASEROUNDROLEID"))
	BotOperatorRoleID = os.Getenv("BOTOPERATORROLEID")
	TwitterEmoji = fmt.Sprintf("%s:%s", os.Getenv("TWITTEREMOJINAME"), os.Getenv("TWITTEREMOJIID"))
	TwitterEmojiName = os.Getenv("TWITTEREMOJINAME")
	return nil
}
