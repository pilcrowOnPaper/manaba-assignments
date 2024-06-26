package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/jedib0t/go-pretty/v6/table"
)

var USERNAME, PASSWORD string

func main() {
	loadEnv()
	client := &http.Client{
		// Avoid automatic redirects.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	authenticatedClient := authenticate(client)
	homeResponse, _ := authenticatedClient.Get("https://room.chuo-u.ac.jp/ct/home_course?chglistformat=list")
	homePage, _ := goquery.NewDocumentFromReader(homeResponse.Body)
	courseIds := homePage.Find("span.courselist-title").Map(func(_ int, selection *goquery.Selection) string {
		href, _ := selection.Children().Attr("href")
		return href
	})
	wg := sync.WaitGroup{}
	assignmentsChannel := make(chan Assignment)
	for _, courseId := range courseIds {
		wg.Add(1)
		go func() {
			courseResponse, _ := authenticatedClient.Get(fmt.Sprintf("https://room.chuo-u.ac.jp/ct/%s_report", courseId))
			courseAssignments := parseAssignmentsFromReportPage(courseResponse)
			for _, assignment := range courseAssignments {
				assignmentsChannel <- assignment
			}
			wg.Done()
		}()
		wg.Add(1)
		go func() {
			courseResponse, _ := authenticatedClient.Get(fmt.Sprintf("https://room.chuo-u.ac.jp/ct/%s_query", courseId))
			courseAssignments := parseAssignmentsFromTestPage(courseResponse)
			for _, assignment := range courseAssignments {
				assignmentsChannel <- assignment
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(assignmentsChannel)
	}()

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Course", "Name", "Deadline"})

	for assignment := range assignmentsChannel {
		t.AppendRow(table.Row{assignment.Course, assignment.Name, assignment.Deadline})
	}
	t.Render()
}

func parseAssignmentsFromReportPage(response *http.Response) []Assignment {
	assignments := []Assignment{}
	reportPage, _ := goquery.NewDocumentFromReader(response.Body)
	courseName := reportPage.Find("#coursename").Text()
	rowsA := reportPage.Find(".stdlist .row1")
	rowsB := reportPage.Find(".stdlist .row")
	rowsA.Each(func(_ int, row *goquery.Selection) {
		state := row.Children().First().Next().Text()
		if strings.Contains(state, "受付中") && strings.Contains(state, "未提出") {
			assignment := Assignment{}
			assignment.Course = courseName
			assignment.Name = row.Children().AttrOr("title", "")
			assignment.Deadline = row.Children().Next().Next().Next().First().Text()
			assignments = append(assignments, assignment)
		}
	})
	rowsB.Each(func(_ int, row *goquery.Selection) {
		state := row.Children().First().Next().Text()
		if strings.Contains(state, "受付中") && strings.Contains(state, "未提出") {
			assignment := Assignment{}
			assignment.Course = courseName
			assignment.Name = row.Children().AttrOr("title", "")
			assignment.Deadline = row.Children().Next().Next().Next().First().Text()
			assignments = append(assignments, assignment)
		}
	})
	return assignments
}

func parseAssignmentsFromTestPage(response *http.Response) []Assignment {
	assignments := []Assignment{}
	reportPage, _ := goquery.NewDocumentFromReader(response.Body)
	courseName := reportPage.Find("#coursename").Text()
	rowsA := reportPage.Find(".stdlist .row0")
	rowsB := reportPage.Find(".stdlist .row1")
	rowsA.Each(func(_ int, row *goquery.Selection) {
		state := row.Children().First().Next().Text()
		if strings.Contains(state, "受付中") && strings.Contains(state, "未提出") {
			assignment := Assignment{}
			assignment.Course = courseName
			assignment.Name = row.Children().First().Children().First().Children().First().Next().Next().Text()
			assignment.Deadline = row.Children().First().Next().Next().Next().Text()
			assignments = append(assignments, assignment)
		}
	})
	rowsB.Each(func(_ int, row *goquery.Selection) {
		state := row.Children().First().Next().Text()
		if strings.Contains(state, "受付中") && strings.Contains(state, "未提出") {
			assignment := Assignment{}
			assignment.Course = courseName
			assignment.Name = row.Children().First().Children().First().Children().First().Next().Next().Text()
			assignment.Deadline = row.Children().First().Next().Next().Next().Text()
			assignments = append(assignments, assignment)
		}
	})
	return assignments
}

type Assignment struct {
	Course   string
	Name     string
	Deadline string
}
