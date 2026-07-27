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
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN environment variable is missing!")
	}

	// 3 Minutes Timeout for large video uploads
	client := &http.Client{
		Timeout: 3 * time.Minute,
	}

	pref := tele.Settings{
		Token:  botToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
		Client: client,
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle("/start", func(c tele.Context) error {
		return c.Send("👋 Welcome! Instagram reel athava YouTube link ayachu tharoo.\nNjan direct video aayi ayachu tharam! 🚀")
	})

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
				tempFileName := fmt.Sprintf("insta_%d.mp4", time.Now().UnixNano())

				// AUTO CLEANUP
				defer func() {
					if _, err := os.Stat(tempFileName); err == nil {
						os.Remove(tempFileName)
						log.Println("Insta Temp file deleted:", tempFileName)
					}
				}()

				err := downloadFile(tempFileName, videoURL)
				if err != nil {
					b.Edit(msg, "❌ Insta video file download cheyyan pattiyilla.")
					return nil
				}

				// Send Video Without Caption
				video := &tele.Video{
					File: tele.FromDisk(tempFileName),
				}

				_, sendErr := b.Send(c.Recipient(), video)
				if sendErr != nil {
					log.Println("Send Error:", sendErr)
					b.Edit(msg, "❌ Video Telegram-ilekk send cheyyan kazhinjilla.")
				} else {
					b.Delete(msg)
				}

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
				tempFileName := fmt.Sprintf("yt_%d.mp4", time.Now().UnixNano())

				// AUTO CLEANUP
				defer func() {
					if _, err := os.Stat(tempFileName); err == nil {
						os.Remove(tempFileName)
						log.Println("YT Temp file deleted:", tempFileName)
					}
				}()

				err := downloadFile(tempFileName, ytData.Result.URL)
				if err != nil {
					b.Edit(msg, "❌ File server-ilekk download cheyyan pattiyilla.")
					return nil
				}

				// Send Video Without Caption
				video := &tele.Video{
					File: tele.FromDisk(tempFileName),
				}

				_, sendErr := b.Send(c.Recipient(), video)
				if sendErr != nil {
					log.Println("Send Error:", sendErr)
					b.Edit(msg, "❌ Video Telegram-ilekk send cheyyan kazhinjilla.")
				} else {
					b.Delete(msg)
				}

			} else {
				b.Edit(msg, "❌ YouTube video fetch cheyyaan kazhinjilla.")
			}
			return nil
		}

		return c.Send("Please Instagram athava YouTube link maathram ayakkoo!")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Bot is live and running!")
		})
		log.Printf("Dummy HTTP Server listening on port %s...", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Printf("HTTP Server Error: %v", err)
		}
	}()

	log.Println("Go Telegram Bot is running...")
	b.Start()
}

func downloadFile(filepath string, url string) error {
	client := &http.Client{Timeout: 2 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
