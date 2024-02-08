package shipyard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

var ErrNotFound = errors.New("todo list not found")

var KindDvmShipyard = 5905
var KindRecommend = 31989

type Config struct {
	Nsec   string   `json:"nsec"`
	Relays []string `json:"relays"`
}

func loadConfig() (*Config, error) {

	env, ok := os.LookupEnv("NOSTR")
	if !ok {
		log.Fatalln("NOSTR env var not set")
	}

	data, err := os.ReadFile(env)
	if err != nil {
		log.Fatalf("Config file: %v", err)
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	return &cfg, nil
}

func findShipyard(ctx context.Context, cfg *Config) error {

	pool := nostr.NewSimplePool(ctx)

	filter := nostr.Filter{
		Kinds: []int{KindRecommend},
		Tags: nostr.TagMap{
			"d": []string{fmt.Sprintf("%d", KindDvmShipyard)},
		},
	}

	ev := pool.QuerySingle(ctx, cfg.Relays, filter)
	if ev == nil {
		return ErrNotFound
	}

	fmt.Println(ev)

	return nil
}

func jobRequest(ctx context.Context, cfg *Config, kind int, content string, time nostr.Timestamp) error {

	var sk string
	var pub string
	if _, s, err := nip19.Decode(cfg.Nsec); err == nil {
		sk = s.(string)
		if pub, err = nostr.GetPublicKey(s.(string)); err != nil {
			return err
		}
	} else {
		return err
	}

	pubEvent := nostr.Event{
		Kind:      kind,
		PubKey:    pub,
		Content:   content,
		CreatedAt: time,
		Tags: nostr.Tags{
			{"t", "dvm"},
			{"t", "shipyard"},
		},
	}
	err := pubEvent.Sign(sk)
	if err != nil {
		return err
	}

	e := nostr.Event{
		Kind:      KindDvmShipyard,
		PubKey:    pub,
		Content:   "",
		CreatedAt: nostr.Now(),
		Tags: nostr.Tags{
			{"i", pubEvent.String(), "text"},
		},
	}
	err = e.Sign(sk)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, r := range cfg.Relays {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()

			relay, err := nostr.RelayConnect(ctx, url)
			if err != nil {
				log.Println(err)
				return
			}
			defer relay.Close()

			err = relay.Publish(ctx, e)
			if err != nil {
				log.Println(err)
				return
			}
		}(r)
	}
	wg.Wait()

	log.Printf("Job request publish with event ID: %s\n", e.ID)

	return nil
}

func Main() error {

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	args := os.Args[1:]

	ctx := context.Background()

	// 	err = findShipyard(ctx, cfg)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	ts, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	kind, err := strconv.Atoi(args[1])
	if err != nil {
		return err
	}

	err = jobRequest(ctx, cfg,
		kind,
		args[2],
		nostr.Timestamp(ts),
	)
	if err != nil {
		return err
	}

	return nil
}
