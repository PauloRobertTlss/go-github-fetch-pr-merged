package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
	"log"
	"os"
	"reflect"
	"strings"
	"time"
)

var (
	COMPANY      = ""
	REPO         = ""
	ACCESS_TOKEN = ""
	MAINTAINEDBY = [8]string{}
	MONTHS       = 3
)

func main() {
	log.Println("started ... ")
	fmt.Println("started github search 90 day")

	//created new .csv
	path, _ := os.Getwd()
	now := time.Now()
	fileoriginalfull := path + "/dataset/" + now.UTC().String() + ".csv"
	logFile, err := os.OpenFile(fileoriginalfull, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ACCESS_TOKEN},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	optPR := &github.PullRequestListOptions{State: "closed"}
	var datasetPR []*github.PullRequest
	last90Days := now.AddDate(0, -MONTHS, 0)

	for {
		fmt.Printf("Fetch page %d\n", optPR.Page)
		pullclosed, resp, err := client.PullRequests.List(ctx, COMPANY, REPO, optPR)
		if err != nil {
			fmt.Printf("Ops! %s", err)
			return
		}
		datasetPR = append(datasetPR, pullclosed...)
		lastPRInsidePage := pullclosed[len(pullclosed)-1]
		createdAt := TimestampToUTC(lastPRInsidePage.GetCreatedAt())
		if createdAt.Before(last90Days) {
			fmt.Printf("Success last %s\n", last90Days.String())
			break //force
		}
		optPR.Page = resp.NextPage
	}

	countAllowed := 0
	opens := 0
	headers := "url;login;created_at;merged_at;duration;total_minutes"
	logFile.Write([]byte(headers))
	logFile.WriteString("\n")

	for _, pr := range datasetPR {
		createdAt := TimestampToUTC(pr.GetCreatedAt())
		user := pr.GetUser()
		merged := TimestampToUTC(pr.GetMergedAt())
		commits := pr.GetMergeCommitSHA()
		timeElapsed := merged.Sub(createdAt)

		if createdAt.After(last90Days) {
			opens++
		}

		//ignore maintained
		if createdAt.After(last90Days) && !itemExists(MAINTAINEDBY, user.GetLogin()) && len(commits) > 0 && !mergeCanceled(timeElapsed.Minutes()) {
			countAllowed++
			row := []string{
				pr.GetHTMLURL(),
				user.GetLogin(),
				pr.GetCreatedAt().String(),
				pr.GetMergedAt().String(),
				timeElapsed.String(),
				fmt.Sprintf("%.0f", timeElapsed.Minutes()),
			}

			logFile.WriteString(strings.Join(row, ";"))
			logFile.WriteString("\n")
		}
	}
	fmt.Println("-----------------------------------------------------------")
	fmt.Println("Total PR's in period:", opens, " | Contributions others users: ", countAllowed)
	log.Println("end")
}

func mergeCanceled(minutes float64) bool {
	return minutes <= 0
}

func itemExists(arrayType interface{}, item interface{}) bool {
	arr := reflect.ValueOf(arrayType)
	if arr.Kind() != reflect.Array {
		panic("Invalid data-type")
	}
	for i := 0; i < arr.Len(); i++ {
		if arr.Index(i).Interface() == item {
			return true
		}
	}

	return false
}

func TimestampToUTC(datetime github.Timestamp) time.Time {
	year, m, d := datetime.Date()
	return time.Date(year, m, d, datetime.Hour(), datetime.Minute(), 0, 0, time.UTC)
}
