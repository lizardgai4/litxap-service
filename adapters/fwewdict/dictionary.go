package fwewdict

import (
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	fwew_lib "github.com/fwew/fwew-lib/v5"
	"github.com/gissleh/litxap"
	"github.com/gissleh/litxap/litxaputil"
)

type fwewDict struct {
	mu sync.Mutex
}

var once sync.Once
var globalDict = &fwewDict{}

func (d *fwewDict) LookupEntries(word string) ([]litxap.Entry, error) {
	stopped := int32(0)

	d.mu.Lock()
	if len(word) > 16 {
		go func() {
			time.Sleep(time.Millisecond * 250)
			if atomic.CompareAndSwapInt32(&stopped, 0, 1) {
				panic("fwew hung on " + word)
			}
		}()
	}

	res, err := fwew_lib.TranslateFromNaviHash(word, true)
	if atomic.CompareAndSwapInt32(&stopped, 0, 1) {
		d.mu.Unlock()
	}
	if err != nil {
		return nil, err
	}

	entries := make([]litxap.Entry, 0, len(res))

	for _, matches := range res {
		for _, match := range matches {
			if match.ID == "" {
				continue
			}

			syllables := strings.Split(strings.ToLower(match.Syllables), "-")

			for _, ipa := range strings.Split(match.IPA, "or") {
				ipa = strings.Trim(ipa, " []")
				ipaSyllables := strings.Split(ipa, ".")
				if len(ipaSyllables) != len(syllables) {
					continue
				}

				stressIndex := 0
				for i, syllable := range ipaSyllables {
					if strings.HasPrefix(syllable, "ˈ") {
						stressIndex = i
						break
					}
				}

				suffixes := append([]string{}, match.Affixes.Suffix...)

				slices.Reverse(suffixes)

				entries = append(entries, litxap.Entry{
					Word:        match.Navi,
					Translation: match.EN,
					Syllables:   syllables,
					Stress:      stressIndex,
					InfixPos:    litxaputil.InfixPositionsFromBrackets(match.InfixLocations, syllables),
					Prefixes:    match.Affixes.Prefix,
					Infixes:     match.Affixes.Infix,
					Suffixes:    suffixes,
				})
			}
		}
	}

	return entries, nil
}

func Adpositions() ([]string, error) {
	list, err := fwew_lib.List([]string{"pos", "has", "adp."}, 0)
	if err != nil {
		return nil, err
	}

	res := make([]string, 0, len(list))
	for _, match := range list {
		res = append(res, match.Navi)
	}

	return res, nil
}

func Global() litxap.Dictionary {
	once.Do(func() {
		fwew_lib.StartEverything()
	})

	return globalDict
}
