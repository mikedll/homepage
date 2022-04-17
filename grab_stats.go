
package main

import (
	"log"
	"os"
	"io/ioutil"
	"github.com/joho/godotenv"
	"encoding/json"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func fileExists(filename string) bool {
    info, err := os.Stat(filename)
    if os.IsNotExist(err) {
        return false
    }
    return !info.IsDir()
}

type CharacterResponse struct {
	Name string `json:"name"`	
}

type StatsResponse struct {
	Character  CharacterResponse `json:"character"`
}

func main() {
	if(fileExists(".env")) {
		loadErr := godotenv.Load()
		if loadErr != nil {
			log.Fatal("Error loading .env file")
		}
	}

	oauth2Conf := &clientcredentials.Config{
		ClientID:     os.Getenv("BNET_CLIENT_ID"),
		ClientSecret: os.Getenv("BNET_CLIENT_SECRET"),
		TokenURL:     "https://us.battle.net/oauth/token",
	}

	client := oauth2Conf.Client(oauth2.NoContext)

	response, err := client.Get("https://us.api.blizzard.com/profile/wow/character/aegwynn/niktonian/achievements/statistics?namespace=profile-us&locale=en_US")
	if err != nil {
		log.Fatal("Got error when retrieving stats")
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Error when parsing body" + err.Error())
	}

	statsResponse := StatsResponse{}
	json.Unmarshal(responseBody, &statsResponse)
	
	log.Println(statsResponse.Character.Name);

	log.Println("Here we go")
}
