package bot

import (
	"fmt"
	"strings"

	"github.com/readysetliqd/airdrop-discord-bot-go/config"

	"github.com/bwmarrin/discordgo"
)

var BotId string

func Start() *discordgo.Session {
	goBot, err := discordgo.New("Bot " + config.Token)

	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	u, err := goBot.User("@me")

	if err != nil {
		fmt.Println(err.Error())
	}

	BotId = u.ID

	goBot.AddHandler(messageHandler)

	err = goBot.Open()

	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	fmt.Println("Bot is running!")
	return goBot
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}

	// if m.content contains botid (Mentions) and "ping" then send "pong!"
	switch {
	case m.Content == "<@"+BotId+"> ping":
		_, _ = s.ChannelMessageSend(m.ChannelID, m.Content)
	case m.Content == config.BotPrefix+"ping":
		_, _ = s.ChannelMessageSend(m.ChannelID, "pong!")
	case strings.Contains(m.Content, "ok"):
		_, _ = s.ChannelMessageSend(m.ChannelID, "OK")
	}
}
