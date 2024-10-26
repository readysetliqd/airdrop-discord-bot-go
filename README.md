# airdrop-discord-bot-go
Airdrop hunting bot for discord to assist pinging roles

Setup Instructions

Setup your own bot instance at discord's dev page
Give the bot permissions TODO which permissions?
Add admin role to discord that will be able to interact with the bot
Add roles to discord that will be tagged for specific funding round types
Optional add carlbot to your discord to allow users to add/remove their tagged roles easily
Set up Google Custom Search account (free trial should be fine)
Create a custom search that has twitter.com/* as possible base searches
Copy your Google Custom Search CX and api key to googlesecrets.env
Edit the config.json file with your bot's token, role IDs, guild ID, and default channel ID which it will post daily