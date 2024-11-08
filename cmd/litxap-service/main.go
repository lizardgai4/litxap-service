package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	fwew_lib "github.com/fwew/fwew-lib/v5"
	"github.com/gissleh/litxap"
	"github.com/gissleh/litxap-service/adapters/fwewdict"
	"github.com/gissleh/litxap-service/adapters/namedict"
)

func main() {
	dict := fwewdict.Global()

	listenAddr := fmt.Sprintf("%s:%s", os.Getenv("HOST"), os.Getenv("PORT"))
	if listenAddr == ":" {
		listenAddr = ":8081"
	}

	// Calculate the multiword words needed at startup
	// Make sure we have words that must be multiword words
	doubles := map[string]string{}
	multis := fwew_lib.GetMultiwordWords()
	for key, val := range multis {
		res, err := fwew_lib.TranslateFromNaviHash(key, true)
		if err != nil || len(res[0]) < 2 {
			doubles[key] = val[0][0]
		}
	}

	log.Println("Starting with address:", listenAddr)

	errCh := make(chan error)
	go func() {
		errCh <- http.ListenAndServe(listenAddr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/run" {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Endpoint not found"})
				return
			}

			dict := dict
			if names := r.URL.Query().Get("names"); names != "" {
				dict = litxap.MultiDictionary{dict, namedict.New(strings.Split(names, ",")...)}
			}

			w.Header().Set("Content-Type", "application/json")

			q := r.URL.Query()
			line, err := litxap.RunLine(q.Get("line"), dict, doubles)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"line": line})
		}))
	}()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-sig:
		log.Println("Got signal:", s)
		os.Exit(0)
	case err := <-errCh:
		log.Println("Got error:", err)
		os.Exit(1)
	}
}
