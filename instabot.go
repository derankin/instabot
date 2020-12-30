package instabot

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"time"
	"unsafe"

	"github.com/ahmdrz/goinsta/v2"
)

// Storing user in session
var checkedUser = make(map[string]bool)

// Instabot is a wrapper around everything
type Instabot struct {
	config             Config
	Insta              *goinsta.Instagram
	username, password string
	Results            struct {
		Followed  int
		Liked     int
		Commented int
	}
}

func New(c Config) Instabot {
	return Instabot{config: c}
}

// login will try to reload a previous session, and will create a new one if it can't
func (i *Instabot) Login() {
	err := i.reloadSession()
	if err != nil {
		i.createAndSaveSession()
	}
}

// reloadSession will attempt to recover a previous session
func (i *Instabot) reloadSession() error {

	insta, err := goinsta.Import("./instabot-session")
	if err != nil {
		return errors.New("Couldn't recover the session")
	}

	if insta != nil {
		i.Insta = insta
	}

	slogger.Info("Successfully logged in")
	return nil

}

// Logins and saves the session
func (i *Instabot) createAndSaveSession() {
	insta := goinsta.New(i.config.User.Username, i.config.User.Password)
	i.Insta = insta
	err := i.Insta.Login()
	check(err)

	err = i.Insta.Export("./instabot-session")
	check(err)
	slogger.Info("Created and saved the session")
}

func (i *Instabot) SyncFollowers() {
	following := i.Insta.Account.Following()
	followers := i.Insta.Account.Followers()

	var followerUsers []goinsta.User
	var followingUsers []goinsta.User

	for following.Next() {
		for _, user := range following.Users {
			followingUsers = append(followingUsers, user)
		}
	}
	for followers.Next() {
		for _, user := range followers.Users {
			followerUsers = append(followerUsers, user)
		}
	}

	var users []goinsta.User
	for _, user := range followingUsers {
		// Skip whitelisted users.
		if containsString(i.config.Whitelist, user.Username) {
			continue
		}

		if !containsUser(followerUsers, user) {
			users = append(users, user)
		}
	}
	if len(users) == 0 {
		return
	}
	fmt.Printf("\n%d users are not following you back!\n", len(users))
	answer := getInput("Do you want to review these users ? [yN]")
	if answer != "y" {
		fmt.Println("Not unfollowing.")
		os.Exit(0)
	}

	answerUnfollowAll := getInput("Unfollow everyone ? [yN]")

	for _, user := range users {
		if answerUnfollowAll != "y" {
			answerUserUnfollow := getInputf("Unfollow %s ? [yN]", user.Username)
			if answerUserUnfollow != "y" {
				i.config.Whitelist = append(i.config.Whitelist, user.Username)
				continue
			}
		}

		i.config.Blacklist = append(i.config.Blacklist, user.Username)

		user.Unfollow()

		time.Sleep(6 * time.Second)
	}
}

// Follows a user, if not following already
func (i *Instabot) followUser(user *goinsta.User) {
	slogger.Infof("Following %s", user.Username)
	err := user.FriendShip()
	check(err)
	// If not following already
	if !user.Friendship.Following {
		user.Follow()
		slogger.Info("Followed")
		i.Results.Followed++
		// report[line{tag, "follow"}]++
	} else {
		slogger.Infof("Already following %s", user.Username)
	}
}

func (i *Instabot) LoopTags() {
	for tag, limits := range i.config.Tags {
		i.Results.Commented = 0
		i.Results.Followed = 0
		i.Results.Liked = 0

		i.browseImages(tag, limits)
	}
	// buildReport()
}

// Likes an image, if not liked already
func (i *Instabot) likeImage(image goinsta.Item) {
	// slogger.Info("Liking the picture")
	if !image.HasLiked {
		image.Like()
		i.Results.Liked++
		slogger.Infof("Liked Image, %d Liked", i.Results.Liked)
	} else {
		slogger.Error("Image already liked")
	}
}

func (i *Instabot) browseImages(tag string, limitConf Tag) {
	var j = 0
	for i.Results.Followed < limitConf.Follow || i.Results.Liked < limitConf.Like || i.Results.Commented < limitConf.Comment {
		slogger.Infof("Fetching the list of images for #%s", tag)
		j++

		// Getting all the pictures we can on the first page
		// Instagram will return a 500 sometimes, so we will retry 10 times.
		// Check retry() for more info.
		var images *goinsta.FeedTag
		err := retry(10, 30*time.Second, func() (err error) {
			images, err = i.Insta.Feed.Tags(tag)
			return
		})
		check(err)

		i.goThrough(images, tag, limitConf)

		if i.config.Limits.MaxRetry > 0 && j > i.config.Limits.MaxRetry {
			slogger.Error("Currently not enough images for this tag to achieve goals")
			break
		}
	}
}

// Goes through all the images for a certain tag
func (i *Instabot) goThrough(images *goinsta.FeedTag, tag string, limits Tag) {
	var j = 0

	// do for other too
	for _, image := range images.Images {
		// Exiting the loop if there is nothing left to do
		if i.Results.Followed >= limits.Follow && i.Results.Liked >= limits.Like && i.Results.Commented >= limits.Comment {
			break
		}

		// Skip our own images
		if image.User.Username == i.config.User.Username {
			continue
		}

		// Check if we should fetch new images for tag
		if j >= limits.Follow && j >= limits.Like && j >= limits.Comment {
			break
		}

		// Skip checked user if the flag is turned on
		if checkedUser[image.User.Username] {
			continue
		}

		// Getting the user info
		// Instagram will return a 500 sometimes, so we will retry 10 times.
		// Check retry() for more info.

		var userInfo *goinsta.User
		err := retry(10, 20*time.Second, func() (err error) {
			userInfo, err = i.Insta.Profiles.ByName(image.User.Username)
			return
		})
		check(err)

		followerCount := userInfo.FollowerCount

		//buildLine()

		checkedUser[userInfo.Username] = true
		slogger.Infof("Checking followers of %s - #%s", userInfo.Username, tag)
		slogger.Infof("%s has %d followers", userInfo.Username, followerCount)
		j++

		// Will only follow and comment if we like the picture
		like := followerCount > i.config.Limits.Like.Min && followerCount < i.config.Limits.Like.Max && i.Results.Liked < limits.Like
		follow := followerCount > i.config.Limits.Follow.Min && followerCount < i.config.Limits.Follow.Max && i.Results.Followed < limits.Follow && like
		comment := followerCount > i.config.Limits.Comment.Min && followerCount < i.config.Limits.Comment.Max && i.Results.Commented < limits.Comment && like

		// Checking if we are already following current user and skipping if we do
		skip := false
		following := i.Insta.Account.Following()

		var followingUsers []goinsta.User
		for following.Next() {
			for _, user := range following.Users {
				followingUsers = append(followingUsers, user)
			}
		}

		for _, user := range followingUsers {
			if user.Username == userInfo.Username {
				skip = true
				break
			}
		}

		// Like, then comment/follow
		if !skip {
			if like {
				i.likeImage(image)
				if follow && !containsString(i.config.Blacklist, userInfo.Username) {
					i.followUser(userInfo)
				}
				if comment {
					i.commentImage(image)
				}
			}
		}
		//slogger.Infof("%s done", userInfo.Username)

		// This is to avoid the temporary ban by Instagram
		time.Sleep(20 * time.Second)
	}
}

// Comments an image
func (i Instabot) commentImage(image goinsta.Item) {
	rand.Seed(time.Now().Unix())
	text := i.config.Comments[rand.Intn(len(i.config.Comments))]
	comments := image.Comments
	if comments == nil {
		// monkey patching
		// we need to do that because https://github.com/ahmdrz/goinsta/pull/299 is not in goinsta/v2
		// I know, it's ugly
		newComments := goinsta.Comments{}
		rs := reflect.ValueOf(&newComments).Elem()
		rf := rs.FieldByName("item")
		rf = reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem()
		item := reflect.New(reflect.TypeOf(image))
		item.Elem().Set(reflect.ValueOf(image))
		rf.Set(item)
		newComments.Add(text)
		// end hack!
	} else {
		comments.Add(text)
	}
	slogger.Infow("Commented ", "comment", text)
	i.Results.Commented++
	//report[line{tag, "comment"}]++
}

func (i Instabot) UpdateConfig() {
	/* TODO rewrite using json marshaling
	viper.Set("whitelist", userWhitelist)
	viper.Set("blacklist", userBlacklist)

	err := viper.WriteConfig()
	if err != nil {
		log.Println("Update config file error", err)
		return
	}

	*/

	slogger.Info("Config file updated")
}
