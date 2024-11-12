package data

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type UnprocessedMessages struct {
	M        map[string]UnprocessedMessage
	Modified bool
}

func (this UnprocessedMessages) IsModified() bool {
	if this.Modified {
		return true
	}
	for _, v := range this.M {
		if v.Changed == true {
			return true
		}
	}
	return false
}

func (this UnprocessedMessages) IsAppended() bool {
	for _, v := range this.M {
		if v.New == true {
			return true
		}
	}
	return false
}

type UnprocessedMessage struct {
	Type         MessageType
	ProtocolName string
	Embeds       []*discordgo.MessageEmbed
	ParentMsgID  string
	Start        int
	New          bool `json:"-"`
	Changed      bool `json:"-"`
}

type Protocols struct {
	M        map[string]Protocol
	Modified bool
	Appended bool
}

type Protocol struct {
	Name       string
	TwitterURL string
	New        bool `json:"-"`
}

type Round struct {
	Name       string
	Desc       string
	Stage      string
	Raise      string
	TotalRaise string
	Category   string
	Tier1Funds string
	Tier2Funds string
}

type Resp struct {
	Total int        `json:"total"`
	Data  []RespData `json:"data"`
}

type RespData struct {
	Icon             string    `json:"icon"`
	Name             string    `json:"name"`
	Key              string    `json:"key"`
	Symbol           any       `json:"symbol"`
	Date             time.Time `json:"date"`
	Raise            int       `json:"raise"`
	PublicSalesRaise any       `json:"publicSalesRaise"`
	TotalRaise       any       `json:"totalRaise"`
	Stage            string    `json:"stage"`
	Status           string    `json:"status"`
	Country          any       `json:"country"`
	TopFollowers     []struct {
		ID             int    `json:"id"`
		Name           string `json:"name"`
		Image          string `json:"image"`
		Username       string `json:"username"`
		TwitterScore   int    `json:"twitterScore"`
		RelatedEntity  any    `json:"relatedEntity"`
		FollowersCount int    `json:"followersCount"`
	} `json:"topFollowers"`
	FollowersCount   int `json:"followersCount"`
	TwitterAccountID int `json:"twitterAccountId"`
	TwitterScore     int `json:"twitterScore"`
	Funds            []struct {
		Name     string `json:"name"`
		Key      string `json:"key"`
		Image    string `json:"image"`
		Tier     int    `json:"tier"`
		Type     string `json:"type"`
		Category struct {
			ID   int    `json:"id"`
			Slug string `json:"slug"`
			Name string `json:"name"`
		} `json:"category"`
		TotalInvestments int `json:"totalInvestments"`
	} `json:"funds"`
	Valuation any `json:"valuation"`
	Category  struct {
		Name string `json:"name"`
		Key  string `json:"key"`
	} `json:"category"`
	HasFundingRounds bool      `json:"hasFundingRounds"`
	CreatedAt        time.Time `json:"createdAt"`
}
