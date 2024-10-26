package config

import (
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
)

// ReadConfig reads the environment variables and assigns them into global exported variables
func ReadConfig() error {
	Token = os.Getenv("token")
	BotPrefix = os.Getenv("botPrefix")
	DefaultChannelID = os.Getenv("channelID")
	GuildID = os.Getenv("guildID")
	FundingRoundRoleID = fmt.Sprintf("<@&%s>", os.Getenv("fundingRoundRoleID"))
	EarlyRoundRoleID = fmt.Sprintf("<@&%s>", os.Getenv("earlyRoundRoleID"))
	BinanceRoundRoleID = fmt.Sprintf("<@&%s>", os.Getenv("binanceRoundRoleID"))
	ParadigmRoundRoleID = fmt.Sprintf("<@&%s>", os.Getenv("paradigmRoundRoleID"))
	CoinbaseRoundRoleID = fmt.Sprintf("<@&%s>", os.Getenv("coinbaseRoundRoleID"))
	BotOperatorRoleID = os.Getenv("botOperatorRoleID")
	TwitterEmoji = fmt.Sprintf("%s:%s", os.Getenv("twitterEmojiName"), os.Getenv("twitterEmojiID"))
	TwitterEmojiName = os.Getenv("twitterEmojiName")
	return nil
}
