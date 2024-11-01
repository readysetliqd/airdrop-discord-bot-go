package bot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/readysetliqd/airdrop-discord-bot-go/config"
	"github.com/readysetliqd/airdrop-discord-bot-go/data"
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"

	"github.com/bwmarrin/discordgo"
)

var BotId string
var unprocessedMessages *data.UnprocessedMessages
var protocols *data.Protocols
var s *discordgo.Session

// tickerDelay is the delay for how often the main loop fires that queries
// cryptorank and posts results to the discord channel
const tickerDelay = 24 * time.Hour

var ticker = time.NewTicker(tickerDelay)
var shutdownSignals = make(chan os.Signal, 1)

func init() {
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)
}

// initialize and add logic for slash commands
var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "list-protocols",
			Description: "Lists all protocols' names.",
		},
		{
			Name:        "get-twitter",
			Description: "Sends a link to the stored twitter url if any",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "protocol",
					Description: "Name of protocol, case and other character sensitive",
					Required:    true,
				},
			},
		},
		{
			Name:        "ping",
			Description: "Sends ping to discord bot. Should respond \"Pong!\" if bot is running.",
		},
		{
			Name:        "force-query-and-reset-timer",
			Description: "Forces a query to crypto rank immediately and resets the 24 hour timer to fire at current time.",
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"list-protocols": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			protocolNames := make([]string, len(protocols.M))
			idx := 0
			for _, protocol := range protocols.M {
				protocolNames[idx] = protocol.Name
				idx++
			}
			sort.Strings(protocolNames)

			var content strings.Builder
			for _, name := range protocolNames {
				content.WriteString(name)
				content.WriteString("\n")
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content.String(),
				},
			})
		},
		"get-twitter": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			protocolName := i.ApplicationCommandData().Options[0].StringValue()
			var content string
			// check protocols map for protocol name (string) sent as get-twitter command options
			if protocol, ok := protocols.M[protocolName]; !ok {
				content = "No protocol by that name exists, check spelling."
			} else {
				content = protocol.TwitterURL
			}

			// check if twitter URL has been stored yet
			if content == "" {
				content = "Twitter URL has not been stored for this protocol yet, look for its funding round embed and click the Twitter react."
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
				},
			})
		},
		"ping": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			content := "Pong!"
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
				},
			})
		},
		"force-query-and-reset-timer": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			content := fmt.Sprintf("Forcing query now. Restarting timer to fire every day at %s", time.Now().Format(time.Kitchen))
			queryCryptoRankAndSendEmbeds()
			ticker.Reset(tickerDelay)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
				},
			})
		},
	}
)

func Start() error {
	// load backup files for stored protocols and unprocessed message into maps
	// and pass up to global scope
	protocols = loadProtocols()
	unprocessedMessages = loadUnprocessedMessages()
	var err error

	// create new discord session and assign it up to global variable s
	s, err = discordgo.New("Bot " + config.Token)
	if err != nil {
		return fmt.Errorf("creating new bot | %w", err)
	}
	log.Println("New session created.")

	// get user details of the bot and assign its ID to global variable BotId
	u, err := s.User("@me")
	if err != nil {
		return fmt.Errorf("getting user details @me | %w", err)
	}
	BotId = u.ID

	// add handlers for all events and interactions
	addEventHandlers()

	// open the websocket to discord for this session
	err = s.Open()
	if err != nil {
		log.Fatal(err)
	}

	// register all the global commands to this guild and store them in a list
	// for removal on shutdown
	registeredCommands := addCommands()

	log.Println("Bot is running!")

	// main go routine loop to query cryptorank, send messages to discord, and add new rounds to rounds file
	go func() {
		for range ticker.C {
			queryCryptoRankAndSendEmbeds()
		}
	}()

	// block until graceful shutdown
	<-shutdownSignals
	log.Println("shutdown signal received")
	gracefulShutdown(registeredCommands)
	return nil
}

// addEventHandlers adds all event handlers to the current session ie. reactions,
// messages, interactions etc.
func addEventHandlers() {
	s.AddHandler(reactionHandler)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
}

// queryCryptoRankAndSendEmbeds is a helper function that calls the functions
// to query crypto rank for the previous day and sends the embeds to the discord
// channel of the results. This function is intended to contain all logic for the
// main daily loop of the bot
func queryCryptoRankAndSendEmbeds() {
	start := time.Now().Add(time.Hour * -24).Format("2006-01-02")
	//TODO add err to this function
	respDataStructs := queryCryptoRank(start)
	err := sendFundingRoundsRecapEmbed(respDataStructs, start)
	if err != nil {
		log.Println(err)
		shutdownSignals <- syscall.SIGTERM
	}
	//TODO add err to this function
	rounds := sendIndividualFundingRoundsEmbeds(respDataStructs)
	if len(rounds) > 0 {
		AppendToFile(data.RoundsFileName, rounds)
	}
}

func reactionHandler(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if isBotOperator(m) && m.ChannelID == config.DefaultChannelID {
		if _, ok := unprocessedMessages.M[m.MessageID]; ok {
			switch unprocessedMessages.M[m.MessageID].Type {
			case data.RoundMsg:
				switch m.MessageReaction.Emoji.Name {
				case config.TwitterEmojiName:
					s.MessageReactionsRemoveAll(config.DefaultChannelID, m.MessageID)
					msg, err := s.ChannelMessage(config.DefaultChannelID, m.MessageID)
					if err != nil {
						//TODO: handle this error and prevent from reaching rest of code block
						fmt.Println("error getting message |", err)
					}
					for _, embed := range msg.Embeds { // there should only be one
						name := embed.Title
						// google search for the twitter website of the name
						newMsg, urlEmbeds := googleSearchForName(s, name, 1)
						unprocessedMessages.M[newMsg.ID] = data.UnprocessedMessage{
							Type:         data.GoogleResult,
							ProtocolName: unprocessedMessages.M[m.MessageID].ProtocolName,
							Embeds:       *urlEmbeds,
							Start:        1,
							ParentMsgID:  m.MessageID,
							New:          true,
						}

						// delete original rounds message from memory
						if !unprocessedMessages.M[m.MessageID].New {
							unprocessedMessages.Modified = true
						}
						delete(unprocessedMessages.M, m.MessageID)

						sendGoogleSearchReacts(s, newMsg, 1)
					}

				default: // do nothing

				}
			case data.GoogleResult:
				switch m.MessageReaction.Emoji.Name {
				case "1️⃣", "2️⃣", "3️⃣":
					// update url field for protocol in memory with twitter url stored in unprocessed messages embeds slice
					var twitterUrl string
					switch m.MessageReaction.Emoji.Name {
					case "1️⃣":
						twitterUrl = unprocessedMessages.M[m.MessageID].Embeds[0].URL
					case "2️⃣":
						twitterUrl = unprocessedMessages.M[m.MessageID].Embeds[1].URL
					case "3️⃣":
						twitterUrl = unprocessedMessages.M[m.MessageID].Embeds[2].URL
					}
					name := unprocessedMessages.M[m.MessageID].ProtocolName
					tmpProtocol := protocols.M[name]
					tmpProtocol.TwitterURL = twitterUrl
					protocols.M[name] = tmpProtocol
					protocols.Modified = true
					s.MessageReactionsRemoveAll(config.DefaultChannelID, m.MessageID)
					s.ChannelMessageDelete(config.DefaultChannelID, m.MessageID)
					if !unprocessedMessages.M[m.MessageID].New {
						unprocessedMessages.Modified = true
					}
					delete(unprocessedMessages.M, m.MessageID)

				case "⬅️":
					if unprocessedMessages.M[m.MessageID].Start > 1 {
						googleSearchNextPage(s, m, -3)
					}

				case "➡️":
					if unprocessedMessages.M[m.MessageID].Start < 97 {
						googleSearchNextPage(s, m, 3)
					}

				case "❌":
					// add parent message back to unprocessed messages map
					ogID := unprocessedMessages.M[m.MessageID].ParentMsgID
					oldMsg, err := s.ChannelMessage(config.DefaultChannelID, ogID)
					if err != nil {
						// TODO handle this error and prevent reaching delete from map and delete google search msg
						// ChannelMessage has built-in retries, discordgo.ErrJSONUnmarshal is returned on any errors during unmarshalling
						fmt.Println("error getting message |", err)
					}
					unprocessedMessages.M[ogID] = data.UnprocessedMessage{Type: data.RoundMsg, ProtocolName: unprocessedMessages.M[m.MessageID].ProtocolName, Embeds: oldMsg.Embeds}

					// add twitter reaction back to parent message
					err = s.MessageReactionAdd(config.DefaultChannelID, ogID, config.TwitterEmoji)
					if err != nil {
						log.Println("failed to add reaction back to message id ", ogID, " | ", err)
					}

					// delete google search result message
					err = s.ChannelMessageDelete(config.DefaultChannelID, m.MessageID)
					if err != nil {
						log.Println("deleting message |", err)
					}

					// we deleted an old (not added to memory in this session) unprocessed
					// message so we need to mark 'Modified' true to trigger backup on shutdown
					if !unprocessedMessages.M[m.MessageID].New {
						unprocessedMessages.Modified = true
					}

					delete(unprocessedMessages.M, m.MessageID)

				default: //do nothing

				}
			}
		}
	}
}

// googleSearchNextPage increments the Start value for which query page to
// search the Google query. 'inc' should be -3 or 3 since this program searches
// in threes, -3 is for previous page. It searches for the next page deleting
// the old message from unprocessedMessages and adding the new search results
// message to it
func googleSearchNextPage(s *discordgo.Session, m *discordgo.MessageReactionAdd, inc int) {
	tmpUnproMsg := unprocessedMessages.M[m.MessageID]
	tmpUnproMsg.Start += inc
	newMsg, urlEmbeds := googleSearchForName(s, tmpUnproMsg.ProtocolName, tmpUnproMsg.Start)
	tmpUnproMsg.Embeds = *urlEmbeds
	tmpUnproMsg.Changed = true
	s.ChannelMessageDelete(config.DefaultChannelID, m.MessageID)
	if !unprocessedMessages.M[m.MessageID].New {
		unprocessedMessages.Modified = true
	}
	delete(unprocessedMessages.M, m.MessageID)
	unprocessedMessages.M[newMsg.ID] = tmpUnproMsg
	sendGoogleSearchReacts(s, newMsg, unprocessedMessages.M[newMsg.ID].Start)
}

// sendGoogleSearchReacts is a helper function that sends predetermined reacts
// to a google search message in the discord channel. It skips sending a left arrow
// if the 'start' of the query is at the first page and skips sending a right
// arrow if the 'start' is at the last page
func sendGoogleSearchReacts(s *discordgo.Session, msg *discordgo.Message, start int) {
	s.MessageReactionAdd(config.DefaultChannelID, msg.ID, "1️⃣")
	s.MessageReactionAdd(config.DefaultChannelID, msg.ID, "2️⃣")
	s.MessageReactionAdd(config.DefaultChannelID, msg.ID, "3️⃣")
	if start > 1 {
		s.MessageReactionAdd(config.DefaultChannelID, msg.ID, "⬅️")
	}
	if start < 97 {
		s.MessageReactionAdd(config.DefaultChannelID, msg.ID, "➡️")
	}
	s.MessageReactionAdd(config.DefaultChannelID, msg.ID, "❌")
}

// googleSearchForName sends a google search query for 'name' starting at query
// number 'start'. it embeds the results and sends them to the discord session
// default channel id then returns the pointer to the new embed message and
// the slice of url embeds
func googleSearchForName(s *discordgo.Session, name string, start int) (*discordgo.Message, *[]*discordgo.MessageEmbed) {
	ctx := context.Background()
	svc, err := customsearch.NewService(ctx, option.WithAPIKey(os.Getenv("GOOGLE_API_KEY")))
	if err != nil {
		log.Fatal("error creating custom search service |", err)
	}
	resp, err := svc.Cse.List().Cx(os.Getenv("GOOGLE_CX")).Q(name).Start(int64(start)).Num(3).OrTerms("web3").OrTerms("crypto").Do()
	if err != nil {
		log.Fatal("error doing search |", err)
	}
	urlEmbeds := []*discordgo.MessageEmbed{}
	for i, result := range resp.Items {
		newEmbed := &discordgo.MessageEmbed{URL: result.FormattedUrl, Title: fmt.Sprintf("#%s %s", strconv.Itoa(i+1), result.Title), Description: result.Snippet}
		urlEmbeds = append(urlEmbeds, newEmbed)
	}
	selectMsg, err := s.ChannelMessageSendEmbeds(config.DefaultChannelID, urlEmbeds)
	if err != nil {
		log.Fatal("error sending message |", err)
	}
	return selectMsg, &urlEmbeds
}

// isBotOperator returns true if the member role for the person that sent this
// message matches the BotOperatorRoleId stored in configs, otherwise returns false
func isBotOperator(m *discordgo.MessageReactionAdd) bool {
	for _, role := range m.Member.Roles {
		if role == config.BotOperatorRoleID {
			return true
		}
	}
	return false
}

// TODO add function description
func sendIndividualFundingRoundsEmbeds(respDataStructs *[]data.RespData) map[string]data.Round {
	// loop over new funding rounds and send an embed to discord for each
	rounds := make(map[string]data.Round, len(*respDataStructs))
	for _, entry := range *respDataStructs {
		time.Sleep(time.Millisecond * 400) // there seems to be some rate limiting for discord messages sent over the bot
		desc := ""
		// Put the coin ticker in the description if one exists
		sym, ok := entry.Symbol.(string)
		if ok {
			desc = sym
		}
		// create slices for fund names and append any tier 1/2 funds respectively
		var tier1 []string
		var tier2 []string
		for _, fund := range entry.Funds {
			switch fund.Tier {
			case 1:
				tier1 = append(tier1, fund.Name)
			case 2:
				tier2 = append(tier2, fund.Name)
			}
		}
		tier1Joined := strings.Join(tier1, ", ")
		tier2Joined := strings.Join(tier2, ", ")
		raise := raiseToString(entry.Raise)
		totalRaise := raiseToString(entry.TotalRaise)
		newEmbed := &discordgo.MessageEmbed{
			Title:       entry.Name,
			Description: desc,
			Color:       16753920,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: entry.Icon,
			},
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Stage", Value: entry.Stage, Inline: true},
				{Name: "Raise", Value: raise, Inline: true},
				{Name: "Total Raise", Value: totalRaise, Inline: true},
				{Name: "Category", Value: entry.Category.Name},
				{Name: "Tier 1 Funds", Value: tier1Joined},
				{Name: "Tier 2 Funds", Value: tier2Joined},
			},
		}

		discordMsg, err := s.ChannelMessageSendEmbed(config.DefaultChannelID, newEmbed)
		if err != nil {
			//TODO return err
			log.Println(err)
		} else {
			newRef := &discordgo.MessageReference{
				MessageID: discordMsg.ID,
				ChannelID: config.DefaultChannelID,
				GuildID:   config.GuildID,
			}
			// check for big names in tier 1 funds and tag roles
			if len(tier1) > 0 {
				for _, fundName := range tier1 {
					if stringContainsCaseIns(fundName, "Binance") {
						s.ChannelMessageSendReply(config.DefaultChannelID, config.BinanceRoundRoleID, newRef)
					}
					if stringContainsCaseIns(fundName, "Coinbase") {
						s.ChannelMessageSendReply(config.DefaultChannelID, config.CoinbaseRoundRoleID, newRef)
					}
					if stringContainsCaseIns(fundName, "Paradigm") {
						s.ChannelMessageSendReply(config.DefaultChannelID, config.ParadigmRoundRoleID, newRef)
					}
				}
			}
			// tag early role if round is seed/pre-seed/extended seed stage
			if stringContainsCaseIns(entry.Stage, "Seed") {
				s.ChannelMessageSendReply(config.DefaultChannelID, config.EarlyRoundRoleID, newRef)
			}
			newRound := data.Round{
				Name:       entry.Name,
				Desc:       desc,
				Stage:      entry.Stage,
				Raise:      raise,
				TotalRaise: totalRaise,
				Category:   entry.Category.Name,
				Tier1Funds: tier1Joined,
				Tier2Funds: tier2Joined,
			}
			rounds[newRef.MessageID] = newRound
			if _, ok := protocols.M[entry.Name]; !ok {
				//TODO after this logic moved to bot package, add new protocol names to command Choices and re-sort
				newProtocol := data.Protocol{Name: entry.Name, New: true}
				protocols.M[entry.Name] = newProtocol
				protocols.Appended = true
			}
			if protocols.M[entry.Name].TwitterURL == "" {
				unprocessedMessages.M[newRef.MessageID] = data.UnprocessedMessage{Type: data.RoundMsg, ProtocolName: entry.Name, Embeds: []*discordgo.MessageEmbed{newEmbed}, New: true}
				err = s.MessageReactionAdd(config.DefaultChannelID, newRef.MessageID, config.TwitterEmoji)
				if err != nil {
					//TODO return err
					log.Println(err)
				}
			}
		}
	}
	return rounds
}

// sendFundingRoundsRecapEmbed sends the message of all funding rounds from today as an embed to discord
func sendFundingRoundsRecapEmbed(respDataStructs *[]data.RespData, start string) error {
	// builds a slice of fields for the embed, a funding round in each field
	fields := make([]*discordgo.MessageEmbedField, len(*respDataStructs))
	for i, data := range *respDataStructs {
		fields[i] = &discordgo.MessageEmbedField{}
		fields[i].Name = data.Name
		sym, ok := data.Symbol.(string)
		if ok {
			fields[i].Name += " - " + sym
		}
		fields[i].Value = fmt.Sprintf("Raise: %s | Stage: %s | Category: %s", raiseToString(data.Raise), data.Stage, data.Category.Name)
	}
	var empty bool
	descr := "All rounds"
	if len(fields) == 0 {
		empty = true
		descr = "None"
	}
	// build and send an embed with a list of all funding rounds
	allRoundsEmbed := &discordgo.MessageEmbed{
		Title:       "Yesterday's Funding Rounds",
		Description: descr,
		Timestamp:   start,
		Color:       8421504,
		Fields:      fields,
		Footer:      &discordgo.MessageEmbedFooter{},
	}
	discordMsg, err := s.ChannelMessageSendEmbed(config.DefaultChannelID, allRoundsEmbed)
	if err != nil {
		return fmt.Errorf("sending embed to channel %s | %w", config.DefaultChannelID, err)
	}
	newRef := &discordgo.MessageReference{
		MessageID: discordMsg.ID,
		ChannelID: config.DefaultChannelID,
		GuildID:   config.GuildID,
	}

	// ping funding rounds role id if there were any funding rounds today
	if !empty {
		_, err = s.ChannelMessageSendReply(config.DefaultChannelID, config.FundingRoundRoleID, newRef)
		if err != nil {
			return fmt.Errorf("sending message repply (tag funding round role) to channel %s | %w", config.DefaultChannelID, err)
		}
	}

	return nil
}

// queryCryptoRank sends an http POST request to cryptorank's API for the previous day's funding rounds
func queryCryptoRank(start string) *[]data.RespData {
	// query the cryptorank.io api for funding rounds within one day
	end := time.Now().Format("2006-01-02")
	body := []byte(`{"limit":20,"filters":{"date":{"start":"` + start + `","end":"` + end + `"}},"skip":0,"sortingColumn":"date","sortingDirection":"DESC"}`)
	//TODO error handling throughout; do something besides panic
	r, err := http.NewRequest("POST", data.PostURL, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	r.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}
	msg, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var resp data.Resp
	err = json.Unmarshal(msg, &resp)
	if err != nil {
		panic(data.JsonMarshalError{OriginalErr: err})
	}
	err = res.Body.Close()
	if err != nil {
		panic(err)
	}
	respDataStructs := resp.Data
	sort.Slice(respDataStructs, func(i, j int) bool {
		return respDataStructs[i].Raise > respDataStructs[j].Raise
	})

	return &respDataStructs
}

// gracefulShutdown backs up both jsonl files for protocols and unprocessed
// messages to disk. It checks if the struct in memory has been modified (a field
// has been changed or a key deleted) or appended (no fields changed only new
// key added). If it has been modified it overwrites the file, if it has been
// appended it appends only the new data to the file. Otherwise it leaves the
// file alone. It logs any errors encountered. It also removes commands from
// the current session's guild that were added on startup.
func gracefulShutdown(registeredCommands *[]*discordgo.ApplicationCommand) {
	// backup unprocessed messages
	if unprocessedMessages.IsModified() {
		err := OverwriteFile(data.UnprocessedMessagesFileName, unprocessedMessages.M)
		if err != nil {
			log.Printf("overwriting file %s | %v\n", data.UnprocessedMessagesFileName, err)
		}
	} else if unprocessedMessages.IsAppended() {
		// build map of only new messages to append
		newMessages := map[string]data.UnprocessedMessage{}
		for name, msg := range unprocessedMessages.M {
			if msg.New {
				newMessages[name] = msg
			}
		}
		err := AppendToFile(data.UnprocessedMessagesFileName, newMessages)
		if err != nil {
			log.Printf("appending to file %s | %v\n", data.UnprocessedMessagesFileName, err)
		}
	}

	// backup protocols
	if protocols.Modified {
		err := OverwriteFile(data.ProtocolsFileName, protocols.M)
		if err != nil {
			log.Printf("overwriting file %s | %v\n", data.ProtocolsFileName, err)
		}
	} else if protocols.Appended {
		// build map of only new protocols to append
		newProtocols := map[string]data.Protocol{}
		for name, protocol := range protocols.M {
			if protocol.New {
				newProtocols[name] = protocol
			}
		}
		err := AppendToFile(data.ProtocolsFileName, newProtocols)
		if err != nil {
			log.Printf("appending to file %s | %v\n", data.ProtocolsFileName, err)
		}
	}

	removeCommands(registeredCommands)
}

// removeCommands is a helper function to remove all registered commands from
// the guild that were added in this current session. Intended to be called on shutdown
func removeCommands(registeredCommands *[]*discordgo.ApplicationCommand) {
	log.Println("Removing commands...")
	for _, v := range *registeredCommands {
		err := s.ApplicationCommandDelete(s.State.User.ID, config.GuildID, v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}
}

// addCommands is a helper function to add commands to the current session from
// global variable commands
func addCommands() *[]*discordgo.ApplicationCommand {
	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, config.GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}
	return &registeredCommands
}

// AppendToFile makes a number of attempts to call tryAppendFile. If it succeeds
// appending the data to file without error, it returns immediately. If an error
// occurs, it retries up to data.MaxWriteAttempts number of calls. If all write
// attempts fail, it tries writing the data to a backup file and returns an error
func AppendToFile[T any](fileName string, dataIn map[string]T) error {
	// return early if nothing to write
	if len(dataIn) == 0 {
		return nil
	}
	var err error
	for attempt := 0; attempt < data.MaxWriteAttempts; attempt++ {
		err = tryAppendFile(fileName, dataIn)
		if err == nil {
			return nil
		} else {
			if !data.IsTemporary(err) {
				return fmt.Errorf("non-temporary error when appending file | %w", err)
			}
		}
	}

	backupFileName := fileName + ".backup"
	backupErr := tryAppendFile(backupFileName, dataIn)
	if backupErr != nil {
		return fmt.Errorf("failed to append file and failed backup data | %w, %w", err, backupErr)
	}
	return fmt.Errorf("data written to backup file at path \"%s\" | %w", backupFileName, err)

}

// tryAppendFile accepts any map type with a string key and opens a jsonl file
// at path fileName and attempts to append the 'dataIn' line by line to the
// end of the file. If at any point besides json marshalling it encounters an
// error it returns a 'temporary' type error.
func tryAppendFile[T any](fileName string, dataIn map[string]T) error {
	var err error
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return data.ReadWriteFileError{OriginalErr: err}
	}

	defer func() {
		cerr := file.Close()
		if err == nil {
			err = cerr
		}
	}()

	// create new writer for file
	writer := bufio.NewWriter(file)

	// write incoming data to file
	for k, v := range dataIn {
		// build a new map to preserve the key when writing entries on
		// their own lines in the .jsonl file
		newMap := map[string]T{k: v}
		jsonNewData, err := json.Marshal(newMap)
		if err != nil {
			return data.JsonMarshalError{OriginalErr: err}
		}
		_, err = writer.WriteString(string(jsonNewData) + "\n")
		if err != nil {
			return data.ReadWriteFileError{OriginalErr: err}
		}
		err = writer.Flush()
		if err != nil {
			return data.ReadWriteFileError{OriginalErr: err}
		}
	}

	return err
}

// OverwriteFile works with any map[string]type. It attempts overwriting the file
// at path 'fileName' with the map passed to 'dataIn'. If any 'temporary' type
// errors are encountered, it retries overwriting up to 'MaxOverwriteAttempts'
// number of times. If all ovewrite attempts fail, it tries writing the data to
// a backup file of the same name. It returns an error if data was written to
// a backup file whether successful or not, or when non temporary error occurs.
func OverwriteFile[T any](fileName string, dataIn map[string]T) error {
	// return early if nothing to write
	if len(dataIn) == 0 {
		return nil
	}
	var err error
	for attempt := 0; attempt < data.MaxWriteAttempts; attempt++ {
		err = tryOverwriteFile(fileName, dataIn)
		if err == nil {
			return nil
		}
		if !data.IsTemporary(err) {
			return fmt.Errorf("non-temporary error when overwriting file | %w", err)
		}
		time.Sleep(data.WriteRetryDelay)
	}

	// failed overwrite too many times, attempt writing to backup file
	backupFileName := fmt.Sprintf("%s.backup", fileName)
	backupErr := tryOverwriteFile(backupFileName, dataIn)
	if backupErr != nil {
		return fmt.Errorf("failed both overwrite to file and writing backup | %w, %w", err, backupErr)
	}

	return fmt.Errorf("data written to backup file \"%v\" | %w", backupFileName, err)
}

// tryOverwriteFile is a helper function for the OverwriteFile function. It
// opens a temporary jsonl file, writes the 'dataIn' to it line by line,
// and then overwrite the file at path 'fileName' with the newly written temporary
// file. Any errors encountered except for during json marshalling are returned
// as a 'temporary' type error.
func tryOverwriteFile[T any](fileName string, dataIn map[string]T) (err error) {
	tmpFileName := "tmp_" + fileName
	tmp, err := os.OpenFile(tmpFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return data.ReadWriteFileError{OriginalErr: err}
	}

	defer func() {
		cerr := tmp.Close()
		// ignore file already closed errors since we're calling Close twice
		if cerr != nil && errors.Is(cerr, os.ErrClosed) {
			cerr = nil
		}
		if err == nil {
			err = cerr
		}
		// we should pass this conditional if any errors encountered either
		// throughout the entire function or from the Close method call
		if err != nil {
			os.Remove(tmpFileName)
		}
	}()

	// write data to jsonl file line by line
	writer := bufio.NewWriter(tmp)
	for k, v := range dataIn {
		bufMap := map[string]T{k: v}
		jsonNewP, err := json.Marshal(bufMap)
		if err != nil {
			return data.JsonMarshalError{OriginalErr: err}
		}
		_, err = writer.WriteString(fmt.Sprintf("%s\n", string(jsonNewP)))
		if err != nil {
			return data.ReadWriteFileError{OriginalErr: err}
		}
		err = writer.Flush()
		if err != nil {
			return data.ReadWriteFileError{OriginalErr: err}
		}
	}

	// sync file, this function is called only on graceful shutdown so we aren't
	// worried with performance
	err = tmp.Sync()
	if err != nil {
		return data.ReadWriteFileError{OriginalErr: err}
	}

	// close tmp file
	err = tmp.Close()
	if err != nil {
		return data.ReadWriteFileError{OriginalErr: err}
	}

	// overwrite file at 'fileName'
	err = os.Rename(tmpFileName, fileName)
	if err != nil {
		return data.ReadWriteFileError{OriginalErr: err}
	}
	return err
}

// loadUnprocessedMessages loads the unprocessed mesages .jsonl file into memory
// parsing the data into a map[string]data.UnprocessedMessage struct and returns it
// it shuts down the program if any errors encountered
func loadUnprocessedMessages() *data.UnprocessedMessages {
	//TODO do something besides quit on errors, implement retries? check for backup file if data is corrupted?
	unprocessedMessagesFile, err := os.OpenFile(data.UnprocessedMessagesFileName, os.O_CREATE|os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal("error opening unprocessedMessagesFile")
	}
	upm := map[string]data.UnprocessedMessage{}
	unpMsgsDecoder := json.NewDecoder(unprocessedMessagesFile)
	for {
		unprocessedMessageId := map[string]data.UnprocessedMessage{}
		err = unpMsgsDecoder.Decode(&unprocessedMessageId)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal("error decoding message |", err)
		}
		for k, v := range unprocessedMessageId {
			upm[k] = v
		}
	}
	err = unprocessedMessagesFile.Close()
	if err != nil {
		log.Fatal("error closing unprocessedMessagesFile")
	}
	ret := &data.UnprocessedMessages{
		M:        upm,
		Modified: false,
	}
	return ret
}

// loadProtocols reads the protocols.jsonl file and parses the data into
// memory. it creates a new data.Protocols struct and assigns the data into
// its map field. It shuts down the program if any errors
func loadProtocols() *data.Protocols {
	//TODO do something besides quit on errors, implement retries? check for backup file if data is corrupted?
	protocolsFile, err := os.OpenFile(data.ProtocolsFileName, os.O_CREATE|os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal("error opening protocols file |", err)
	}
	protocols := map[string]data.Protocol{}
	protocolsDecoder := json.NewDecoder(protocolsFile)
	for {
		protocol := map[string]data.Protocol{}
		err = protocolsDecoder.Decode(&protocol)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Fatal("error decoding protocols file |", err)
			}
		}
		for k, v := range protocol {
			protocols[k] = v
		}
	}
	err = protocolsFile.Close()
	if err != nil {
		log.Fatal("error closing protocols file |", err)
	}
	p := &data.Protocols{M: protocols, Modified: false}
	return p
}

// stringContainsCaseIns is a helper function to compare a string s and substring
// in a case insensitive manner
func stringContainsCaseIns(s, substring string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substring))
}

// raiseToString converts an int to a readable formatted string for either
// millions or thousands and returns it. Returns "N/A" if the int is 0
func raiseToString(raise int) string {
	switch {
	case raise == 0:
		return "N/A"
	case raise < 1000000:
		return fmt.Sprintf("$%.2fK", float32(raise)/1000)
	default:
		return fmt.Sprintf("$%.2fM", float32(raise)/1000000)
	}
}
