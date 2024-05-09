package main

import (
	"encoding/binary"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/bwmarrin/discordgo"
)

const RENATO_ID = "277619169195982850"
const RAVI_ID = "99667684139995136"
const TILTALO_ID = "336231733416427522"
const LUIZ_ID = "381161936231858176"
const SAMUEL_ID = "155503310785347584"
const JVOL_ID = "332528404027015168"
const VITOR_ID = "362436507543535616"
const KAUHAN_ID = "939672698928365578"

var buffer = make([][]byte, 0)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		slog.Error("failed to load env variables", "error", err)
	}

	if os.Getenv("DISCORD_BOT_TOKEN") == "" {
		slog.Info("No token provided. Please run: airhorn -t <bot token>")
		return
	}

	// Load the sound file.
	err := loadSound()
	if err != nil {
		slog.Error("Error loading sound: ", "error", err)
		slog.Info("Please copy $GOPATH/src/github.com/bwmarrin/examples/airhorn/airhorn.dca to this directory.")
		return
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		slog.Error("Error creating Discord session: ", "error", err)
		return
	}

	// Register ready as a callback for the ready events.
	dg.AddHandler(ready)

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(messageCreate)

	// Register guildCreate as a callback for the guildCreate events.
	dg.AddHandler(guildCreate)

	dg.AddHandler(userJoinedChannel)

	// We need information about guilds (which includes their channels),
	// messages and voice states.
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent |
		discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuildVoiceStates

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		slog.Error("Error opening Discord session: ", "error", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	slog.Info("Airhorn is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {
	slog.Info("Ready Running")
	// Set the playing status.
	s.UpdateGameStatus(0, "!airhorn")
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	slog.Info("MessageCreate Running")
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	slog.Info("Message Author", "id", m.Author.ID, "username", m.Author.Username, "botID", s.State.User.ID)
	if m.Author.ID == s.State.User.ID {
		return
	}

	// check if the message is "!airhorn"
	slog.Info("Message Content", "content", m.Content, "has prefix", strings.HasPrefix(m.Content, "!airhorn"))
	if strings.HasPrefix(m.Content, "!airhorn") {
		slog.Info("Should play airhorn")
		// Find the channel that the message came from.
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}

		// Find the guild for that channel.
		g, err := s.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}

		// Look for the message sender in that guild's current voice states.
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				err = playSound(s, g.ID, vs.ChannelID)
				if err != nil {
					slog.Error("Error playing sound:", "error", err)
				}

				return
			}
		}
	}
}

// This function will be called (due to AddHandler above) every time a new
// guild is joined.
func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	slog.Info("GuildCreate Running")
	if event.Guild.Unavailable {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			_, _ = s.ChannelMessageSend(channel.ID, "Airhorn is ready! Type !airhorn while in a voice channel to play a sound.")
			return
		}
	}
}

func userJoinedChannel(s *discordgo.Session, event *discordgo.VoiceStateUpdate) {
	slog.Info("UserJoinedChannel Running")
	if event.UserID == s.State.User.ID {
		return
	}

	targetID := RENATO_ID

	if event.UserID == targetID {
		g, err := s.State.Guild(event.GuildID)
		if err != nil {
			slog.Error("Could not find guild")
			return
		}

		membersInChannel := getMembersInChannel(g, event.ChannelID)
		if len(membersInChannel) <= 2 {
			return
		}

		// otherVoiceChannel := getOtherVoiceChannel(g, event.ChannelID)
		// if otherVoiceChannel == "" {
		// 	slog.Error("Failed to find new channel")
		// 	return
		// }

		// for _, member := range membersInChannel {
		// 	if member != targetID {
		// 		if err := s.GuildMemberMove(event.GuildID, member, &otherVoiceChannel); err != nil {
		// 			slog.Error("Failed to move member", "error", err)
		// 			return
		// 		}
		// 	}
		// }

		newChannel, err := createChannelSecret(s, g, targetID)
		if err != nil {
			slog.Error("Failed to create new channel", "error", err)
			return
		}

		for _, member := range membersInChannel {
			if member != targetID {
				if err := s.GuildMemberMove(event.GuildID, member, &newChannel); err != nil {
					slog.Error("Failed to move member", "error", err)
				}
			}
		}
	}
}

func getMembersInChannel(g *discordgo.Guild, channelId string) []string {
	var members []string
	for _, vs := range g.VoiceStates {
		if vs.ChannelID == channelId {
			members = append(members, vs.UserID)
		}
	}
	slog.Info("Members In Channel", "members", members, "channel", channelId)
	return members
}

func getOtherVoiceChannel(g *discordgo.Guild, channelId string) string {
	for _, channel := range g.Channels {
		if channel.ID != channelId && channel.Type == discordgo.ChannelTypeGuildVoice {
			slog.Info("New Target Channel", "name", channel.Name, "type", channel.Type)
			return channel.ID
		}
	}
	return ""
}

func createChannelSecret(s *discordgo.Session, g *discordgo.Guild, memberID string) (string, error) {
	antiUserChannelData := discordgo.GuildChannelCreateData{
		Name: "Anti-Zhurma",
		Type: discordgo.ChannelTypeGuildVoice,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{
				ID:   memberID,
				Type: discordgo.PermissionOverwriteTypeMember,
				Deny: discordgo.PermissionViewChannel | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceSpeak,
			},
		},
	}
	c, err := s.GuildChannelCreateComplex(g.ID, antiUserChannelData, func(cfg *discordgo.RequestConfig) {
		cfg.MaxRestRetries = 0
		cfg.ShouldRetryOnRateLimit = false
	})
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

// loadSound attempts to load an encoded sound file from disk.
func loadSound() error {

	file, err := os.Open("./airhorn.dca")
	if err != nil {
		slog.Error("Error opening dca file :", "error", err)
		return err
	}

	var opuslen int16

	for {
		// Read opus frame length from dca file.
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		// If this is the end of the file, just return.
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err := file.Close()
			if err != nil {
				return err
			}
			return nil
		}

		if err != nil {
			slog.Error("Error reading from dca file :", "error", err)
			return err
		}

		// Read encoded pcm from dca file.
		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		// Should not be any end of file errors
		if err != nil {
			slog.Error("Error reading from dca file :", "error", err)
			return err
		}

		// Append encoded pcm data to the buffer.
		buffer = append(buffer, InBuf)
	}
}

// playSound plays the current buffer to the provided channel.
func playSound(s *discordgo.Session, guildID, channelID string) (err error) {
	return nil // TODO: Remove

	// Join the provided voice channel.
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(250 * time.Millisecond)

	// Start speaking.
	vc.Speaking(true)

	// Send the buffer data.
	for _, buff := range buffer {
		vc.OpusSend <- buff
	}

	// Stop speaking
	vc.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(250 * time.Millisecond)

	// Disconnect from the provided voice channel.
	vc.Disconnect()

	return nil
}
