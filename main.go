package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/kkdai/youtube/v2"
)

func main() {
	token := loadToken()

	sess, err := discordgo.New(token)

	if err != nil {
		log.Fatal(err)
	}

	sess.AddHandler(onMessageCreate)

	sess.Identify.Intents = discordgo.IntentsAll

	err = sess.Open()

	if err != nil {
		log.Fatal(err)
	}

	defer sess.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func loadToken() string {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading env variables %s", err)
	}

	token := os.Getenv("TOKEN")

	return token
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate)  {
		if m.Author.ID == s.State.User.ID || m.Author.Bot {
			return
		}

		if !strings.HasPrefix(m.Content, "!play") {
			return
		}

		youtubeURL := strings.TrimSpace(strings.TrimPrefix(m.Content, "!play"))

		err := playFromYouTubeURL(s, m.GuildID, m.Author.ID, youtubeURL)
		if err != nil {
			fmt.Println("Error playing song: ", err)
			return
		}
}

func connectToVoiceChannel(s *discordgo.Session, guildID, userID string) (*discordgo.VoiceConnection, error) {
	vs, err := s.State.VoiceState(guildID, userID)
	if err != nil {
		return nil, err
	}

	vc, err := s.ChannelVoiceJoin(guildID, vs.ChannelID, false, false)
	if err != nil {
		return nil, err
	}

	return vc, nil
}

func playFromYouTubeURL(s *discordgo.Session, guildID, userID, url string) error {
	vc, err := connectToVoiceChannel(s, guildID, userID)
	if err != nil {
		return err
	}
	defer vc.Disconnect()

	audioFile, err := downloadYouTubeAudio(url)
	if err != nil {
		fmt.Println("Error in audioFile", err)
		return err
	}
	defer os.Remove(audioFile)

	vc.Speaking(true)

	dgvoice.PlayAudioFile(vc, audioFile, make(chan bool))

	vc.Speaking(false)

	return nil
}

func downloadYouTubeAudio(url string) (string, error) {
	videoID := url
	client := youtube.Client{}

	video, err := client.GetVideo(videoID)
	if err != nil {
		panic(err)
	}

	formats := video.Formats.WithAudioChannels() // only get videos with audio
	stream, _, err := client.GetStream(video, &formats[0])
	fmt.Println(stream)
	if err != nil {
		panic(err)
	}

	file, err := os.Create("video.mp3")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		panic(err)
	}

	return file.Name(), err
}

