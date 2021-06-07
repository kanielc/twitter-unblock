package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

type Credentials struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

const BlockPath = "https://api.twitter.com/1.1/blocks"

func main() {
	at := flag.String("at", "", "Access Token")
	as := flag.String("as", "", "Access Token Secret")
	ck := flag.String("ck", "", "Consumer Key")
	cs := flag.String("cs", "", "Consumer Secret")
	flag.Parse()

	rand.Seed(time.Now().Unix())

	if *at == "" || *as == "" || *ck == "" || *cs == "" {
		flag.Usage()
		os.Exit(0)
	}

	// todo: Remove these
	creds := Credentials{
		AccessToken:       *at,
		AccessTokenSecret: *as,
		ConsumerKey:       *ck,
		ConsumerSecret:    *cs,
	}

	config := oauth1.NewConfig(creds.ConsumerKey, creds.ConsumerSecret)
	oathToken := oauth1.NewToken(creds.AccessToken, creds.AccessTokenSecret)
	httpClient := config.Client(context.Background(), oathToken)
	client := twitter.NewClient(httpClient)

	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}

	user, _, err := client.Accounts.VerifyCredentials(verifyParams)
	checkError(err)
	log.Printf("Logged in - User's Name: %s\tHandle: %s\tID: %d", user.Name, user.ScreenName, user.ID)

	blocks, err := GetBlockedIds(httpClient)
	checkError(err)

	// Parameterize how many to unblock
	chosen := blocks[rand.Intn(len(blocks))]
	blockedUsers, _, err := client.Users.Lookup(&twitter.UserLookupParams{UserID: []int64{chosen}})
	checkError(err)

	blockedUser := blockedUsers[0]
	fmt.Printf("Now unblocking: User's Name: %s\tHandle: %s\tID: %d\n", blockedUser.Name, blockedUser.ScreenName, blockedUser.ID)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/destroy.json?user_id=%d", BlockPath, chosen), nil)
	_, err = httpClient.Do(req)

	checkError(err)
}

type BlockResponse struct {
	Ids        []int64 `json:"ids"`
	NextCursor int     `json:"next_cursor"`
}

func GetBlockedIds(client *http.Client) ([]int64, error) {
	ids := make([]int64, 0)
	var err error
	nextCursor := 0

	for err == nil && nextCursor >= 0 {
		var cursorParam string

		if nextCursor > 0 {
			cursorParam = fmt.Sprintf("?cursor=%d", nextCursor)
		}

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/ids.json%s", BlockPath, cursorParam), nil)
		req.Header.Add("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return ids, err
		}

		defer resp.Body.Close()

		var blocks BlockResponse
		json.NewDecoder(resp.Body).Decode(&blocks)
		if n := blocks.NextCursor; n == 0 {
			nextCursor = -1
		} else {
			nextCursor = n
		}

		ids = append(ids, blocks.Ids...)
	}

	return ids, nil
}
