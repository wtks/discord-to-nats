package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
)

var (
	discordBotToken    = os.Getenv("DISCORD_BOT_TOKEN")
	natsUrl            = os.Getenv("NATS_URL")
	publishTopicPrefix = os.Getenv("PUBLISH_TOPIC_PREFIX")
	messagePrefix      = os.Getenv("MESSAGE_PREFIX")
)

var isStringAlphabetic = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString

func main() {
	nc, err := nats.Connect(natsUrl)
	if err != nil {
		log.Fatal(err)
	}
	c, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	dg, err := discordgo.New("Bot " + discordBotToken)
	if err != nil {
		log.Fatal(err)
	}
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions)
	dg.AddHandler(messageCreateHandler(c))

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func messageCreateHandler(c *nats.EncodedConn) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, msg *discordgo.MessageCreate) {
		if msg.Author.ID == s.State.User.ID {
			return
		}
		if msg.Author.Bot {
			return
		}
		if !strings.HasPrefix(msg.Content, messagePrefix) {
			return
		}

		args := strings.Fields(strings.TrimPrefix(msg.Content, messagePrefix))
		if len(args) == 0 {
			return
		}
		if !isStringAlphabetic(args[0]) {
			return
		}

		payload := map[string]interface{}{
			"message_id":  msg.ID,
			"channel_id":  msg.ChannelID,
			"user_id":     msg.Author.ID,
			"raw_content": msg.Content,
			"args":        args,
			"args_len":    len(args),
		}

		if err := c.Publish(publishTopicPrefix+"."+args[0], payload); err != nil {
			log.Error(err)
			return
		}

		if err := s.MessageReactionAdd(msg.ChannelID, msg.ID, "📩"); err != nil {
			log.Error(err)
			return
		}
	}
}
