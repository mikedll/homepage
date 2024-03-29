
package main

import (
	"log"
	"os"
	"io/ioutil"
	"fmt"
	"regexp"
	"time"
	"strings"
	"net/http"
	"sort"
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
	Total int
	Counts []DungeonFinishedCount
}

type AchievementCategory struct {
  Name string `json:name`
  Id int      `json:id`
}

type AchievementCategoriesResponse struct {
  Categories []AchievementCategory   `json:categories`
}

type Character struct {
  Id    int      `json:id`
  Name  string   `json:name`
}
type Account struct {
  Characters  []Character  `json:characters`
}

type AccountResponse struct {
  Accounts []Account    `json:wow_accounts`
}

func listAchievementCategories(client *http.Client) {
  // https://us.api.blizzard.com/data/wow/achievement-category/index?namespace=static-us&locale=en_US&access_token=US3sTZMJz8d7yhTSnslcYTTNnuBVvjMSvJ
	response, err := client.Get("https://us.api.blizzard.com/data/wow/achievement-category/index" + 
    "?namespace=static-us&locale=en_US")
	if err != nil {
		log.Fatal("Got error when retrieving stats")
	}
  
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Error while reading body" + err.Error())
	}
	// log.Println(string(responseBody))
  
  if(response.StatusCode >= 300) {
    log.Printf("Got bad status code when retrieving achievement categories: %d", response.StatusCode);
    log.Printf("%s", responseBody)
    log.Fatal("Exiting")
  }
  
  achievementCategoriesResponse := AchievementCategoriesResponse{}
  err = json.Unmarshal(responseBody, &achievementCategoriesResponse)
  if(err != nil) {
    log.Fatal("Failed to unmarshal achievement categories response")
  }
  
  for _, cat := range achievementCategoriesResponse.Categories {
    if strings.Contains(cat.Name, "Dungeons & Raids") {
      log.Printf("%d, %s", cat.Id, cat.Name)      
    } 
  }
}

func listCharacters(config *clientcredentials.Config) {
  var err error
  myClient := http.Client{}
  var response *http.Response
  response, err = myClient.Get("https://us.api.blizzard.com/profile/user/wow?namespace=profile-us&locale=en_US&access_token=TOKEN_NEEDED")
	if err != nil {
		log.Fatal("Got error when retrieving stats")
	}
  
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Error while reading body" + err.Error())
	}
	log.Println(string(responseBody))
  
  if(response.StatusCode >= 300) {
    log.Printf("Got bad status code when retrieving account info (like characters): %d", response.StatusCode);
    log.Printf("%s", responseBody)
    log.Fatal("Exiting")
  }
  
  accountResponse := AccountResponse{}
  for _, char := range accountResponse.Accounts[0].Characters {
    log.Printf("Name: %s", char.Name)
  }
}

func getCharAllExpsStats(client *http.Client, charName string) ([]ExpansionDungeonStats, time.Time) {
	response, err := client.Get("https://us.api.blizzard.com/profile/wow/character/aegwynn/" + charName + "/achievements/statistics?namespace=profile-us&locale=en_US")
	if err != nil {
		log.Fatal("Got error when retrieving stats")
	}
  
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Error while reading body" + err.Error())
	}
	// log.Println(string(responseBody))
  
  if(response.StatusCode >= 300) {
    log.Printf("Got bad status code: %d", response.StatusCode);
    log.Printf("%s", responseBody)
    log.Fatal("Exiting")
  }

	statsResponse := StatsResponse{}
	json.Unmarshal(responseBody, &statsResponse)
	
	// log.Println("Name: " + statsResponse.Character.Name);

	var dungeonCategory CategoryResponse
	for _, cat := range statsResponse.Categories {
    log.Printf("Name: %s", cat.Name)
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

	// "rolledUp" means special dungeons that have different
	// end bosses, like Assault on Violet Hold, will be merged together.
	rolledUpExpStats := []ExpansionDungeonStats{}

	re := regexp.MustCompile("^(.*) \\((.*)\\)$")
	for _, curExp := range expStats {
		// log.Println(curExp.Name)
		dungeonToCount := make(map[string]int)
		for _, dungeonStats := range curExp.Counts {
			matches := re.FindStringSubmatch(dungeonStats.Description)
			if _, ok := dungeonToCount[matches[2]]; !ok {
				dungeonToCount[matches[2]] = 0
			}
			// log.Println("Found " + matches[2] + " in " + dungeonStats.Description)
			dungeonToCount[matches[2]] += dungeonStats.Quantity
		}
		rolledUpDungeonCounts := []DungeonFinishedCount{}
		for dungeonName, count := range dungeonToCount {
			rolledUpDungeonCounts = append(rolledUpDungeonCounts, DungeonFinishedCount{
				Description: dungeonName,
				Quantity: count,
			})
		}

		totalThisExp := 0
		for _, rolledUpDungeonCount := range rolledUpDungeonCounts {
			totalThisExp += rolledUpDungeonCount.Quantity
		}

		rolledUpExpStats = append(rolledUpExpStats, ExpansionDungeonStats{
			Name: curExp.Name,
			Total: totalThisExp,
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

	time.LoadLocation("America/Los_Angeles")
	
	oauth2Conf := &clientcredentials.Config{
		ClientID:     os.Getenv("BNET_CLIENT_ID"),
		ClientSecret: os.Getenv("BNET_CLIENT_SECRET"),
		TokenURL:     "https://us.battle.net/oauth/token",
	}

	client := oauth2Conf.Client(oauth2.NoContext)
  
  if(len(os.Args) > 1) {
    if(os.Args[1] == "list_acats") {
      listAchievementCategories(client)
    } else if(os.Args[1] == "list_chars") {
      listCharacters(oauth2Conf);
    } else {
      log.Fatal("Unable to determine your request from the command line arguments")
    }
    return;
  }  
  
	charList := strings.Split(os.Getenv("BNET_CHARS"), ",")
	expStatsAry := []ExpansionDungeonStats{}
	var mostRecentAtTime time.Time
	for _, name := range (charList) {
		charAllExpsStats, retrievedMostRecentAtTime := getCharAllExpsStats(client, name)
		if retrievedMostRecentAtTime.After(mostRecentAtTime) {
			mostRecentAtTime = retrievedMostRecentAtTime
		}
		for _, charExpDungeonStats := range charAllExpsStats {
			var matchingExpStats *ExpansionDungeonStats
			for i := range expStatsAry {
				if expStatsAry[i].Name == charExpDungeonStats.Name {
					matchingExpStats = &expStatsAry[i]
					break
				}
			}
			if matchingExpStats == nil {
				expStatsAry = append(expStatsAry, ExpansionDungeonStats{
					Name: charExpDungeonStats.Name,
					Total: 0,
				})
				matchingExpStats = &expStatsAry[len(expStatsAry) - 1]
			}

			// charExpDungeonStats.Counts == [Black Rook Hold, Eye of Ashara, ...]
			for _, charCount := range charExpDungeonStats.Counts {
				found := false
				for i := range (*matchingExpStats).Counts {
					if (*matchingExpStats).Counts[i].Description == charCount.Description {
						found = true
						(*matchingExpStats).Counts[i].Quantity += charCount.Quantity
						(*matchingExpStats).Total += charCount.Quantity
						break
					}
				}
				if !found {
					(*matchingExpStats).Counts = append((*matchingExpStats).Counts, DungeonFinishedCount{
						Description: charCount.Description,
						Quantity: charCount.Quantity,
					})
					(*matchingExpStats).Total += charCount.Quantity
				}
			}

			sort.Slice((*matchingExpStats).Counts, func(i, j int) bool {
				return (strings.Compare((*matchingExpStats).Counts[i].Description, (*matchingExpStats).Counts[j].Description) == -1)
			})
		}
	}

	expNameToOrder := map[string]int{
		"Classic": 1,
		"Burning Crusade": 2,
		"Wrath of the Lich King": 3,
		"Cataclysm": 4,
		"Mists of Pandaria": 5,
		"Warlords of Draenor": 6,
		"Legion": 7,
		"Battle for Azeroth": 8,
		"Shadowlands": 9,
		"Dragonflight": 10,
	}
	
	sort.Slice(expStatsAry, func(i, j int) bool {
		return expNameToOrder[expStatsAry[i].Name] < expNameToOrder[expStatsAry[j].Name]
	})
		
	totalAllExps := 0
	for _, curExp := range expStatsAry {
		totalAllExps += curExp.Total
	}
	
	var body string
	for _, curExp := range expStatsAry {
		for _, dungeonCount := range curExp.Counts {
			body += fmt.Sprintf("%s - %s: %d\n", curExp.Name, dungeonCount.Description, dungeonCount.Quantity)
		}
		body += fmt.Sprintf("%s Total: %d\n", curExp.Name, curExp.Total)
	}
	body += fmt.Sprintf("All Expansions Total: %d\n", totalAllExps)
	
	// fmt.Println(mostRecentAt)
	body += "\nLast Updated: " + mostRecentAtTime.Format("Mon Jan 2, 2006 at 3:04pm MST") + "\n"
	fmt.Printf(body)

	var html []byte
	var err error
	html, err = ioutil.ReadFile("wow.html")
	if err != nil {
		log.Fatal("Unable to read html source")
	}

	wowRe := regexp.MustCompile("{{wow}}")
	readied := wowRe.ReplaceAllString(string(html), body)

  log.Fatal("We no longer write output from this script. Blizzard's API didn't appear to be working when we" +
    "last checked on this script.")
	ioutil.WriteFile("./build-output/wow.html", []byte(readied), 0644)
	
	log.Println("Done")
}
