package data

import "time"

const (
	PostURL = "https://api.cryptorank.io/v0/funding-rounds-v2"
)

const (
	RoundsFileName              = "rounds.jsonl"
	ProtocolsFileName           = "protocols.jsonl"
	UnprocessedMessagesFileName = "unprocessed_messages.jsonl"
	GoogleSecretsEnvFileName    = "googlesecrets.env"
	ConfigEnvFileName           = "config.env"
)

type MessageType int

const (
	RoundMsg MessageType = iota
	GoogleResult
)

const (
	WriteRetryDelay = time.Second * 3
	// MaxOverwriteAttempts is the number of attempts the program will try to overwrite a file in OverwriteFile() function. Includes the initial attempt. Must be 1 or more
	MaxWriteAttempts = 3
)
