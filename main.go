package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type SlackBotRequest struct {
	Type      string `json:"type"`      // Used to validate event type
	Challenge string `json:"challenge"` // Used to challenge slack on url validation
	Token     string `json:"token"`
	TeamID    string `json:"team_id"`
	Event     struct {
		Type    string `json:"type"`
		EventTS string `json:"event_ts"`
		Channel string `json:"channel"`
		Links   []struct {
			Domain string `json:"domain"`
			URL    string `json:"url"`
		}
	} `json:"event"`
}

type SlackBotResponse struct {
	Text string `json:"text"`
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	router := gin.New()
	router.Use(gin.Logger())

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "Pong")
	})

	router.POST("/slack-message", func(c *gin.Context) {

		var slackReq SlackBotRequest
		err := c.BindJSON(&slackReq)
		if err != nil {
			log.Fatal(err)
		}

		// check if slack is trying to check configuration
		if slackReq.Type == "url_verification" {
			fmt.Println(slackReq.Challenge)

			c.String(http.StatusOK, slackReq.Challenge)
			return
		}

		// Find the item in mercado livre URL
		re := regexp.MustCompile(`[A-Z]{3}\-\d+`)
		meliItemID := strings.Replace(re.FindString(slackReq.Event.Links[0].URL), "-", "", -1)
		meliItem := getMeliItem(meliItemID)
		meliItemDescription := getMeliItemDescription(meliItemID)

		var jsonStr = []byte(fmt.Sprintf(`{"channel": "%s","blocks": [{"type": "section", "text": { "type": "mrkdwn", "text": "*%s*\n\n*%s* %v" }, "accessory": { "type": "image", "image_url": "%s", "alt_text": "%s"}},{"type": "context","elements": [{"type": "mrkdwn","text": "*Description:* %s "}]}]}`, slackReq.Event.Channel, meliItem.Title, meliItem.CurrencyID, meliItem.Price, meliItem.SecureThumbnail, meliItem.Title, strings.Replace(meliItemDescription.PlainText, `"`, "'", -1)))

		fmt.Println(string(jsonStr))

		req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Slack-No-Retry", "1")
		req.Header.Set("Authorization", os.Getenv("SLACK_BOT_TOKEN"))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		defer resp.Body.Close()

		c.String(http.StatusOK, "Mensagem enviada.")
	})

	router.Run(":" + port)
}

type MeliItem struct {
	Title           string  `json:"title"`
	Price           float32 `json:"price"`
	CurrencyID      string  `json:"currency_id"`
	SecureThumbnail string  `json:"secure_thumbnail"`
	Permalink       string  `json:"permalink"`
	Descriptions    []struct {
		ID string `json:"id"`
	} `json:"descriptions"`
}

func getMeliItem(meliItem string) (item MeliItem) {
	resp, err := http.Get(fmt.Sprintf(`https://api.mercadolibre.com/items/%s`, meliItem))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&item)
	if err != nil {
		log.Fatal(err)
	}

	return item
}

type MeliItemDescription struct {
	PlainText string `json:"plain_text"`
}

func getMeliItemDescription(meliItem string) (itemDesc MeliItemDescription) {
	resp, err := http.Get(fmt.Sprintf(`https://api.mercadolibre.com/items/%s/description`, meliItem))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&itemDesc)
	if err != nil {
		log.Fatal(err)
	}

	return itemDesc
}
