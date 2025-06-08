package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"regexp/syntax"
	"slices"

	"github.com/gorilla/websocket"
)

//go:embed static
var staticFiles embed.FS

func BuildRoutes(mux *http.ServeMux, manager *ChannelManager, devMode bool) {
	var staticHandler http.Handler
	if devMode {
		staticHandler = http.FileServer(http.Dir("static"))
	} else {
		staticDir, err := fs.Sub(staticFiles, "static")
		if err != nil {
			panic(err)
		}
		staticHandler = http.FileServer(http.FS(staticDir))
	}
	mux.Handle("GET /", staticHandler)
	mux.Handle("GET /api/channels", listChannels(manager))
	mux.Handle("POST /api/channels", createChannel(manager))
	mux.Handle("GET /api/channels/{channelID}/history", channelHistory(manager))
	mux.Handle("GET /api/channels/{channelID}/plain", channelPlain(manager))
	mux.Handle("PATCH /api/channels/{channelID}", updateChannel(manager))
	mux.Handle("DELETE /api/channels/{channelID}", channelDelete(manager))
	mux.Handle("POST /api/validate-filter", http.HandlerFunc(validateFilter))
	mux.Handle("GET /api/channels/{channelID}/live", channelLive(manager))
}

func channelHistory(manager *ChannelManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("channelID")
		channel, err := manager.ChannelByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(channel.History()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func channelPlain(manager *ChannelManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("channelID")
		channel, err := manager.ChannelByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Add("Content-Type", "text/plain")
		w.Header().Add("Content-Disposition", fmt.Sprintf("filename=%q", channel.Name+".txt"))

		for _, line := range channel.History() {
			if line.Event != "" {
				continue
			}
			if _, err := fmt.Fprintln(w, line.Text); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

type listChannelsResponse struct {
	Channels []channelResponse `json:"channels"`
}

type channelResponse struct {
	Name           string `json:"name"`
	ID             string `json:"id"`
	OutputFilename string `json:"outputFilename"`
	Filter         string `json:"filter"`
	Replace        string `json:"replace"`
}

func listChannels(manager *ChannelManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channels := make([]channelResponse, 0, len(manager.Channels))

		for _, channel := range manager.Channels {
			filter := ""
			if channel.Filter != nil {
				filter = channel.Filter.String()
			}
			channels = append(channels, channelResponse{
				Name:           channel.Name,
				ID:             channel.ID,
				OutputFilename: channel.OutputFilename,
				Filter:         filter,
				Replace:        channel.Replace,
			})
		}

		if err := json.NewEncoder(w).Encode(listChannelsResponse{Channels: channels}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func createChannel(manager *ChannelManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		formName := r.FormValue("name")
		formFilter := r.FormValue("filter")
		formReplace := r.FormValue("replace")

		filter, err := regexp.Compile(formFilter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		channel := NewChannel(formName, filter, formReplace)
		if err := manager.AddChannel(channel); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := channelResponse{
			Name:           channel.Name,
			ID:             channel.ID,
			OutputFilename: channel.OutputFilename,
			Filter:         filter.String(),
			Replace:        formReplace,
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func updateChannel(manager *ChannelManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("channelID")
		if id == "stdin" {
			http.Error(w, "cannot modidy stdin", http.StatusBadRequest)
			return
		}

		channel, err := manager.ChannelByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if err := r.ParseMultipartForm(2 * 1024); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if name, ok := r.Form["name"]; ok {
			channel.SetName(name[0])
		}

		if replace, ok := r.Form["replace"]; ok {
			if channel.SetReplace(replace[0]) {
				w.Write([]byte("replace updated\r\n"))
			}
		}

		if filter, ok := r.Form["filter"]; ok {
			if err := channel.SetFilter(filter[0]); err != nil {
				// Don't return wrapping error if it's from the regex parser
				regexErr := &syntax.Error{}
				if errors.As(err, &regexErr) {
					err = regexErr
				}
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Write([]byte("filter updated\r\n"))
		}
	}
}

func validateFilter(w http.ResponseWriter, r *http.Request) {
	filter, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = regexp.Compile(string(filter))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func channelLive(manager *ChannelManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("channelID")
		channel, err := manager.ChannelByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		upgrader := websocket.Upgrader{}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		defer ws.Close()

		broadcast := make(chan Line, 10)
		channel.AddBroadcast(broadcast)
		defer channel.RemoveBroadcast(broadcast)

		for line := range broadcast {
			ws.WriteJSON(line)
		}
	}

}

func channelDelete(manager *ChannelManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("channelID")

		if slices.Contains([]string{"stdin", "stdout"}, id) {
			http.Error(w, "cannot delete channel "+id, http.StatusBadRequest)
			return
		}

		manager.RemoveChannel(id)
	}
}
