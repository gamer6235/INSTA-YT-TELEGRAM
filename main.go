package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

// API Response Structures
type InstaResponse struct {
	Status bool `json:"status"`
	Result []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"result"`
}

type YTResponse struct {
	Status bool `json:"status"`
	Result struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	} `json:"result"`
}

func main() {
	// 1. Fetch Bot Token from Environment Variable
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN environment variable is missing!")
	}

	pref := tele.Settings{
		Token:  botToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	// 2. Start Command Handler
	b.Handle("/start", func(c tele.Context) error {
		return c.Send("👋 Welcome! Instagram reel athava YouTube link ayachu tharoo.\nNjan direct video aayi ayachu tharam! 🚀")
	})

	// 3. Main Text Message Handler
	b.Handle(tele.OnText, func(c tele.Context) error {
		text := c.Text()

		// --- INSTAGRAM LINK HANDLER ---
		if strings.Contains(text, "instagram.com") {
			msg, _ := b.Send(c.Recipient(), "🔄 Instagram video process cheyyunnu...")

			encodedURL := url.QueryEscape(text)
			apiURL := fmt.Sprintf("https://api.nexray.eu.cc/downloader/instagram?url=%s", encodedURL)

			resp, err := http.Get(apiURL)
			if err != nil || resp.StatusCode != 200 {
				b.Edit(msg, "❌ Insta API Connect cheyyaan kazhinjilla.")
				return nil
			}
			defer resp.Body.Close()

			var instaData InstaResponse
			json.NewDecoder(resp.Body).Decode(&instaData)

			if instaData.Status && len(instaData.Result) > 0 {
				videoURL := instaData.Result[0].URL

				// Send Direct Video Stream to Telegram (0 Disk & RAM usage)
				video := &tele.Video{
					File:    tele.FromURL(videoURL),
					Caption: "✨ Downloaded via Insta Downloader",
				}
				b.Send(c.Recipient(), video)
				b.Delete(msg)
			} else {
				b.Edit(msg, "❌ Insta Video link fetch cheyyaan pattiyilla.")
			}
			return nil
		}

		// --- YOUTUBE LINK HANDLER ---
		if strings.Contains(text, "youtube.com") || strings.Contains(text, "youtu.be") {
			msg, _ := b.Send(c.Recipient(), "⏳ YouTube Video download cheyyunnu (480p format)...")

			encodedURL := url.QueryEscape(text)
			apiURL := fmt.Sprintf("https://api.nexray.eu.cc/downloader/v1/ytmp4?url=%s&resolusi=480", encodedURL)

			resp, err := http.Get(apiURL)
			if err != nil || resp.StatusCode != 200 {
				b.Edit(msg, "❌ YT API response tharaan vaiki.")
				return nil
			}
			defer resp.Body.Close()

			var ytData YTResponse
			json.NewDecoder(resp.Body).Decode(&ytData)

			if ytData.Status && ytData.Result.URL != "" {
				// Create unique local temp file name
				tempFileName := fmt.Sprintf("temp_%d.mp4", time.Now().UnixNano())

				// AUTO CLEANUP: Video send aayalum error vannaalum local file auto-delete aakum
				defer func() {
					if _, err := os.Stat(tempFileName); err == nil {
						os.Remove(tempFileName)
						log.Println("Temp file auto-deleted successfully:", tempFileName)
					}
				}()

				// Download file to server disk temporarily
				err := downloadFile(tempFileName, ytData.Result.URL)
				if err != nil {
					b.Edit(msg, "❌ File server-ilekk download cheyyan pattiyilla.")
					return nil
				}

				// Send Video File to Telegram
				video := &tele.Video{
					File:    tele.FromDisk(tempFileName),
					Caption: fmt.Sprintf("🎬 **%s**\n\n✨ Downloaded via YT Bot", ytData.Result.Title),
				}
				b.Send(c.Recipient(), video)
				b.Delete(msg)

			} else {
				b.Edit(msg, "❌ YouTube video fetch cheyyaan kazhinjilla.")
			}
			return nil
		}

		return c.Send("Please Instagram athava YouTube link maathram ayakkoo!")
	})

	// 4. Dummy Web Server for Render Port Check
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Bot is running live!")
		})
		log.Printf("Dummy HTTP Server listening on port %s for Render...", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Printf("HTTP Server Error: %v", err)
		}
	}()

	// 5. Start Telegram Bot Polling
	log.Println("Go Telegram Bot is running...")
	b.Start()
}

// Helper Function: Stream down video file to local temp path
func downloadFile(filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}
