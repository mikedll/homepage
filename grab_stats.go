
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

type SubCatStatisticsResponse struct {
	Id    int     `json:"id"`
	Name  string  `json:"name"`
	Quantity int  `json:"quantity"`
}

type SubCategoryResponse struct {
	Id          int                        `json:"id"`
	Name        string                     `json:"name"`
	Statistics  []SubCatStatisticsResponse `json:"statistics"`
}

type CategoryResponse struct {
	Id            int                   `json:"id"`
	Name          string                `json:"name"`
	SubCategories []SubCategoryResponse `json:"sub_categories"`
}

type StatsResponse struct {
	Character   CharacterResponse  `json:"character"`
	Categories  []CategoryResponse `json:"categories"`
}

type DungeonFinishedCount struct {
	Description string
	Quantity int
}

type ExpansionDungeonStats struct {
	Name  string
	Counts []DungeonFinishedCount
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
	
	log.Println("Name: " + statsResponse.Character.Name);

	var dungeonCategory CategoryResponse
	for _, cat := range statsResponse.Categories {
		if cat.Name == "Dungeons & Raids" {
			dungeonCategory = cat
		}
	}
	
	if dungeonCategory.Name == "" {
		log.Fatal("Didn't find dungeon category")
	}

	var stats []ExpansionDungeonStats = []ExpansionDungeonStats{}
	for _, subCat := range dungeonCategory.SubCategories {
		curStats := ExpansionDungeonStats {
			Name: subCat.Name,
		}
		stats = append(stats, curStats)
	}

	for _, cur := range stats {
		log.Println(cur.Name)
	}
	
	log.Println("Done")
}
