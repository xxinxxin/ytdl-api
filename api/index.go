package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/kkdai/youtube/v2"
	"net/http"
	"net/url"
	"os"
	"time"
	"math/rand"

	_ "github.com/joho/godotenv/autoload"
)

// Fungsi untuk memuat daftar proxy dari file
func loadProxies(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxies = append(proxies, scanner.Text())
	}
	return proxies, scanner.Err()
}

// Fungsi untuk mendapatkan proxy acak dari daftar
func getRandomProxy(proxies []string) string {
	if len(proxies) == 0 {
		return ""
	}
	rand.Seed(time.Now().UnixNano())
	return proxies[rand.Intn(len(proxies))]
}

// Handler untuk menangani permintaan HTTP
func Handler(w http.ResponseWriter, r *http.Request) {
	// Muat daftar proxy dari file
	proxies, err := loadProxies("proxy.txt")
	if err != nil {
		http.Error(w, "Error loading proxies: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Pilih proxy secara acak dari daftar
	Socks5Proxy := getRandomProxy(proxies)

	ytClient := youtube.Client{}
	var client *http.Client
	if Socks5Proxy != "" {
		proxyURL, err := url.Parse(Socks5Proxy)
		if err != nil {
			http.Error(w, "Invalid proxy URL: "+err.Error(), http.StatusInternalServerError)
			return
		}
		client = &http.Client{Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}}
		ytClient = youtube.Client{HTTPClient: client}
	}

	switch r.URL.Path {
	case "/":
		msg := `Welcome to ytDl API
/dl?url=<video_url> - Download a single video
/playlist?url=<playlist_url> - Download a playlist

Example:
/dl?url=https://www.youtube.com/watch?v=video_id
/dl?url=video_id
/playlist?url=https://www.youtube.com/playlist?list=playlist_id
/playlist?url=playlist_id

Made with ❤ by @Abishnoi69
Golang API for downloading YouTube videos and playlists
`
		if Socks5Proxy == "" {
			msg += "No SOCKS5 proxy configured, maybe you get rate limited by YouTube :("
		}
		_, _ = fmt.Fprint(w, msg)

	case "/dl":
		videoURL := r.URL.Query().Get("url")
		if videoURL == "" {
			http.Error(w, "Please provide a video URL", http.StatusBadRequest)
			return
		}

		video, err := ytClient.GetVideo(videoURL)
		if err != nil {
			http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		formats := video.Formats.WithAudioChannels()
		streamURL, err := ytClient.GetStreamURL(video, &formats[0])
		if err != nil {
			http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]string{
			"ID":          video.ID,
			"author":      video.Author,
			"duration":    video.Duration.String(),
			"thumbnail":   video.Thumbnails[0].URL,
			"description": video.Description,
			"stream_url":  streamURL,
			"title":       video.Title,
			"view_count":  fmt.Sprintf("%d", video.Views),
		}

		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Error encoding JSON response: "+err.Error(), http.StatusInternalServerError)
		}

	case "/playlist":
		playlistURL := r.URL.Query().Get("url")
		if playlistURL == "" {
			http.Error(w, "Please provide a playlist URL", http.StatusBadRequest)
			return
		}

		playlist, err := ytClient.GetPlaylist(playlistURL)
		if err != nil {
			http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var videos []map[string]string
		for _, entry := range playlist.Videos {
			video, err := ytClient.VideoFromPlaylistEntry(entry)
			if err != nil {
				http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
				return
			}

			streamURL, err := ytClient.GetStreamURL(video, &video.Formats[0])
			if err != nil {
				http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
				return
			}

			videoInfo := map[string]string{
				"ID":          video.ID,
				"author":      video.Author,
				"duration":    video.Duration.String(),
				"thumbnail":   video.Thumbnails[0].URL,
				"description": video.Description,
				"stream_url":  streamURL,
				"title":       video.Title,
				"view_count":  fmt.Sprintf("%d", video.Views),
			}
			videos = append(videos, videoInfo)
		}

		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(videos); err != nil {
			http.Error(w, "Error encoding JSON response: "+err.Error(), http.StatusInternalServerError)
		}

	default:
		http.NotFound(w, r)
	}
}
