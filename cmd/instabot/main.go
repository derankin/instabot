package main

import (
	"github.com/derankin/instabot"
)



func main() {
	// Gets the command line options
	flags := instabot.ParseFlags()
	// Gets the config
	//instabot.ReadConfig()
	config := instabot.LoadConfiguration("config/config.json")
	// Tries to login
	insta := instabot.New(config)
	insta.Login()
	if flags.Unfollow {
		insta.SyncFollowers()
	} else if flags.Run {
		// Loop through tags ; follows, likes, and comments, according to the config file
		insta.LoopTags()
	}
	insta.UpdateConfig()
}
