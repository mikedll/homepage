
package main

import (
	"log"
	"os"
	"io/ioutil"
	"strconv"
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

	var expStats []ExpansionDungeonStats = []ExpansionDungeonStats{}
	for _, subCat := range dungeonCategory.SubCategories {
		finishedCounts := []DungeonFinishedCount{}
		for _, subCatStats := range subCat.Statistics {
			log.Println("Found quantity of sub cat stats " + strconv.Itoa(subCatStats.Id) + " - " + strconv.Itoa(subCatStats.Quantity))
			finishedCounts = append(finishedCounts, DungeonFinishedCount {
				Description: subCatStats.Name,
				Quantity: subCatStats.Quantity,
			})
		}
		curDungeonStats := ExpansionDungeonStats {
			Name: subCat.Name,
			Counts: finishedCounts,
		}
		expStats = append(expStats, curDungeonStats)
	}

	for _, curExp := range expStats {
		log.Println(curExp.Name)
		for _, dungeonStats := range curExp.Counts {
			log.Println("  " + dungeonStats.Description + ": " + strconv.Itoa(dungeonStats.Quantity))
		}
	}
	
	log.Println("Done")
}