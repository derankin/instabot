package instabot

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ahmdrz/goinsta/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"strings"
	"time"
)

/*
var (
	// Whether we are in development mode or not
	dev bool

	// Whether we want an email to be sent when the script ends / crashes
	nomail bool

	// Whether we want to launch the unfollow mode
	unfollow bool

	// Acut
	run bool

	// Whether we want to have logging
	logs bool

	// Used to skip following, liking and commenting same user in this session
	noduplicate bool
)

// An image will be liked if the poster has more followers than likeMin, and less than likeMax
var likeMin int
var likeMax int

// A user will be followed if he has more followers than followMin, and less than followMax
// Needs to be a subset of the like interval
var followMin int
var followMax int

// An image will be commented if the poster has more followers than commentMin, and less than commentMax
// Needs to be a subset of the like interval
var commentMin int
var commentMax int

// Hashtags list. Do not put the '#' in the config file
var tags map[string]interface{}

// Limits for the current hashtag
var limits map[string]int

// Comments list
var comments []string

// Line is a struct to store one line of the report
type line struct {
	Tag, Action string
}

// Report that will be sent at the end of the script
var report map[line]int

var userBlacklist []string
var userWhitelist []string

// Counters that will be incremented while we like, comment, and follow
var numFollowed int
var numLiked int
var numCommented int

// Will hold the tag value
var tag string

*/
var logger *zap.Logger
var slogger *zap.SugaredLogger

// check will log.Fatal if err is an error
func check(err error) {
	if err != nil {
		slogger.Errorw("an error has occurred ", "errors", err)
	}
}

type Flags struct {
	Run, Unfollow, Nomail, Dev, Nodups bool
}
type Tag struct {
	Comment int
	Like    int
	Follow  int
}

type Config struct {
	Tags     map[string]Tag `json:"tags"`
	Comments []string       `json:"comments"`
	User     struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"user"`
	Limits struct {
		MaxRetry int `json:"maxretry"`
		Comment  struct {
			Max int `json:"max"`
			Min int `json:"min"`
		} `json:"comment"`
		Follow struct {
			Max int `json:"max"`
			Min int `json:"min"`
		} `json:"follow"`
		Like struct {
			Max int `json:"max"`
			Min int `json:"min"`
		} `json:"like"'`
	} `json:"limits"`
	Blacklist []string `json:"blacklist"`
	Whitelist []string `json:"whitelist"`
}

func LoadConfiguration(file string) Config {
	var config Config
	if file == "" {
		file = "config/config.json"
	}

	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		slogger.Errorf("Could not load configuration file. %s", err)
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	slogger.Infof("config: running as [%s]", config.User.Username)
	slogger.Infof("config: max retry [%d]", config.Limits.MaxRetry)
	return config
}

func InitLogger() {
	writerSyncer := getLogWriter()
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writerSyncer, zapcore.DebugLevel)
	//logger = zap.New(core, zap.AddCaller())
	logger = zap.New(core)
	slogger = logger.Sugar()
}

func getLogWriter() zapcore.WriteSyncer {
	lumberjackLogger := &lumberjack.Logger{
		Filename:   "./instabot.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}
	writeSyncers := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(lumberjackLogger))
	return writeSyncers
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// Parses the options given to the script
func ParseFlags() Flags {
	var (
		dev         bool
		nomail      bool
		unfollow    bool
		run         bool
		noduplicate bool
	)
	flag.BoolVar(&run, "run", false, "Use this option to follow, like and comment")
	flag.BoolVar(&unfollow, "sync", false, "Use this option to unfollow those who are not following back")
	flag.BoolVar(&nomail, "nomail", false, "Use this option to disable the email notifications")
	flag.BoolVar(&dev, "dev", false, "Use this option to use the script in development mode : nothing will be done for real")
	flag.BoolVar(&noduplicate, "noduplicate", false, "Use this option to skip following, liking and commenting same user in this session")

	flag.Parse()

	/*
		// -logs enables the log file
		if logs {

				// Opens a log file
				t := time.Now()
				logFile, err := os.OpenFile("instabot-"+t.Format("2006-01-02-15-04-05")+".log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
				check(err)
				defer logFile.Close()

				// Duplicates the writer to stdout and logFile
				mw := io.MultiWriter(os.Stdout, logFile)
				log.SetOutput(mw)

			InitLogger()
		}

	*/

	InitLogger()

	opts := Flags{
		Run:      run,
		Unfollow: unfollow,
		Nomail:   nomail,
		Dev:      dev,
		Nodups:   noduplicate,
	}
	slogger.Infow("Flags parsed", "flags", opts)
	return opts
}

/*
// Reads the conf in the config file
func ReadConfig() {
	folder := "config"
	if dev {
		folder = "local"
	}
	viper.SetConfigFile(folder + "/config.json")

	// Reads the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	// Check environment
	viper.SetEnvPrefix("instabot")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	// Confirms which config file is used
	log.Printf("Using config: %s\n\n", viper.ConfigFileUsed())


	likeMin = viper.GetInt("limits.like.min")
	likeMax = viper.GetInt("limits.like.max")

	followMin = viper.GetInt("limits.follow.min")
	followMax = viper.GetInt("limits.follow.max")

	commentMin = viper.GetInt("limits.comment.min")
	commentMax = viper.GetInt("limits.comment.max")

	tags = viper.GetStringMap("tags")

	comments = viper.GetStringSlice("comments")

	userBlacklist = viper.GetStringSlice("blacklist")
	userWhitelist = viper.GetStringSlice("whitelist")

	report = make(map[line]int)
}



// Sends an email. Check out the "mail" section of the "config.json" file.
func send(body string, success bool) {
	if !nomail {
		from := viper.GetString("user.mail.from")
		pass := viper.GetString("user.mail.password")
		to := viper.GetString("user.mail.to")

		status := func() string {
			if success {
				return "Success!"
			}
			return "Failure!"
		}()
		msg := "From: " + from + "\n" +
			"To: " + to + "\n" +
			"Subject:" + status + "  go-instabot\n\n" +
			body

		err := smtp.SendMail(viper.GetString("user.mail.smtp"),
			smtp.PlainAuth("", from, pass, viper.GetString("user.mail.server")),
			from, []string{to}, []byte(msg))

		if err != nil {
			log.Printf("smtp error: %s", err)
			return
		}

		log.Print("sent")
	}
}

*/

// Retries the same function [function], a certain number of times (maxAttempts).
// It is exponential : the 1st time it will be (sleep), the 2nd time, (sleep) x 2, the 3rd time, (sleep) x 3, etc.
// If this function fails to recover after an error, it will send an email to the address in the config file.
func retry(maxAttempts int, sleep time.Duration, function func() error) (err error) {
	for currentAttempt := 0; currentAttempt < maxAttempts; currentAttempt++ {
		err = function()
		if err == nil {
			return
		}
		for i := 0; i <= currentAttempt; i++ {
			time.Sleep(sleep)
		}
		slogger.Errorw("Retrying after error:", "error", err)
	}

	//send(fmt.Sprintf("The script has stopped due to an unrecoverable error :\n%s", err), false)
	return fmt.Errorf("After %d attempts, last error: %s", maxAttempts, err)
}

/*
// Builds the line for the report and prints it
func buildLine() {
	reportTag := ""
	for index, element := range report {
		if index.Tag == tag {
			reportTag += fmt.Sprintf("%s %d/%d - ", index.Action, element, limits[index.Action])
		}
	}
	// Prints the report line on the screen / in the log file
	if reportTag != "" {
		log.Println(strings.TrimSuffix(reportTag, " - "))
	}
}

// Builds the report prints it and sends it
func buildReport() {
	reportAsString := ""
	for index, element := range report {
		var times string
		if element == 1 {
			times = "time"
		} else {
			times = "times"
		}
		if index.Action == "like" {
			reportAsString += fmt.Sprintf("#%s has been liked %d %s\n", index.Tag, element, times)
		} else {
			reportAsString += fmt.Sprintf("#%s has been %sed %d %s\n", index.Tag, index.Action, element, times)
		}
	}

	// Displays the report on the screen / log file
	fmt.Println(reportAsString)

	// Sends the report to the email in the config file, if the option is enabled
	//send(reportAsString, true)
}

*/

func getInput(text string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(text)
	input, err := reader.ReadString('\n')
	check(err)
	return strings.TrimSpace(input)
}

// Checks if the user is in the slice
func containsUser(slice []goinsta.User, user goinsta.User) bool {
	for _, currentUser := range slice {
		if currentUser.Username == user.Username {
			return true
		}
	}
	return false
}

func getInputf(format string, args ...interface{}) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(format, args...)
	input, err := reader.ReadString('\n')
	check(err)
	return strings.TrimSpace(input)
}

// Same, with strings
func containsString(slice []string, user string) bool {
	for _, currentUser := range slice {
		if currentUser == user {
			return true
		}
	}
	return false
}
