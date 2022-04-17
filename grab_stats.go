
package main

import (
	"log"
	"os"
	"io/ioutil"
	"fmt"
	"regexp"
	"time"
	"net/http"
	// "strconv"
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
	Id          int     `json:"id"`
	Name        string  `json:"name"`
	Quantity    float64 `json:"quantity"`
	LastUpdated int64   `json:"last_updated_timestamp"`
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

func getCharAllExpsStats(client *http.Client, charName string) ([]ExpansionDungeonStats, time.Time) {
	response, err := client.Get("https://us.api.blizzard.com/profile/wow/character/aegwynn/" + charName + "/achievements/statistics?namespace=profile-us&locale=en_US")
	if err != nil {
		log.Fatal("Got error when retrieving stats")
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Error when parsing body" + err.Error())
	}
	// log.Println(string(responseBody))

	statsResponse := StatsResponse{}
	json.Unmarshal(responseBody, &statsResponse)
	
	// log.Println("Name: " + statsResponse.Character.Name);

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
	var mostRecentAt int64
	for _, subCat := range dungeonCategory.SubCategories {
		finishedCounts := []DungeonFinishedCount{}
		for _, subCatStats := range subCat.Statistics {
			finishedCounts = append(finishedCounts, DungeonFinishedCount {
				Description: subCatStats.Name,
				Quantity: int(subCatStats.Quantity),
			})
			if mostRecentAt == 0 || subCatStats.LastUpdated > mostRecentAt {
				mostRecentAt = subCatStats.LastUpdated
			}
		}
		curDungeonStats := ExpansionDungeonStats {
			Name: subCat.Name,
			Counts: finishedCounts,
		}
		expStats = append(expStats, curDungeonStats)
	}
	mostRecentAtTime := time.Unix(mostRecentAt / 1000, 0)

	re := regexp.MustCompile("^(.*) \\((.*)\\)$")
	rolledUpExpStats := []ExpansionDungeonStats{}
	for _, curExp := range expStats {
		// log.Println(curExp.Name)
		dungeonToCount := make(map[string]int)
		for _, dungeonStats := range curExp.Counts {
			matches := re.FindStringSubmatch(dungeonStats.Description)
			if _, ok := dungeonToCount[matches[2]]; !ok {
				dungeonToCount[matches[2]] = 0
			}
			dungeonToCount[matches[2]] += dungeonStats.Quantity
		}
		rolledUpDungeonCounts := []DungeonFinishedCount{}
		for dungeonName, count := range dungeonToCount {
			rolledUpDungeonCounts = append(rolledUpDungeonCounts, DungeonFinishedCount{
				Description: dungeonName,
				Quantity: count,
			})
		}

		rolledUpExpStats = append(rolledUpExpStats, ExpansionDungeonStats{
			Name: curExp.Name,
			Counts: rolledUpDungeonCounts,
		})
	}

	return rolledUpExpStats, mostRecentAtTime
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

	expStatsAry := []ExpansionDungeonStats{}
	var mostRecentAtTime time.Time
	for _, name := range ([]string{"niktonian", "audney"}) {
		retrievedAllExpsStats, retrievedMostRecentAtTime := getCharAllExpsStats(client, name)
		if retrievedMostRecentAtTime.After(mostRecentAtTime) {
			mostRecentAtTime = retrievedMostRecentAtTime
		}
		for _, retrievedDungeonStats := range retrievedAllExpsStats {
			var matchingExpStats *ExpansionDungeonStats
			for i := range expStatsAry {
				if expStatsAry[i].Name == retrievedDungeonStats.Name {
					matchingExpStats = &expStatsAry[i]
					break
				}
			}
			if matchingExpStats == nil {
				expStatsAry = append(expStatsAry, ExpansionDungeonStats{
					Name: retrievedDungeonStats.Name,
				})
				matchingExpStats = &expStatsAry[len(expStatsAry) - 1]
			}

			// retrievedDungeonStats.Counts == [Black Rook Hold, Eye of Ashara, ...]
			for _, retrievedCount := range retrievedDungeonStats.Counts {
				found := false
				for i := range (*matchingExpStats).Counts {
					if (*matchingExpStats).Counts[i].Description == retrievedCount.Description {
						found = true
						(*matchingExpStats).Counts[i].Quantity += retrievedCount.Quantity
						break
					}
				}
				if !found {
					(*matchingExpStats).Counts = append((*matchingExpStats).Counts, DungeonFinishedCount{
						Description: retrievedCount.Description,
						Quantity: retrievedCount.Quantity,
					})
				}
			}
		}
	}

	var body string
	for _, curExp := range expStatsAry {
		for _, dungeonCount := range curExp.Counts {
			body += fmt.Sprintf("%s - %s: %d\n", curExp.Name, dungeonCount.Description, dungeonCount.Quantity)
		}
	}
	
	// fmt.Println(mostRecentAt)
	body += "\nLast Updated: " + mostRecentAtTime.Format("Mon Jan 2, 2006 at 3:04 MST") + "\n"
	fmt.Printf(body)

	var html []byte
	var err error
	html, err = ioutil.ReadFile("index.src.html")
	if err != nil {
		log.Fatal("Unable to read html source")
	}

	wowRe := regexp.MustCompile("{{wow}}")
	readied := wowRe.ReplaceAllString(string(html), body)

	ioutil.WriteFile("index.html", []byte(readied), 0644)
	
	log.Println("Done")
}
