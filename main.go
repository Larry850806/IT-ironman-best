package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
)

func getNumberOfPage(group string) int {
	baseURL := "https://ithelp.ithome.com.tw/ironman/signup/list"
	url := fmt.Sprintf("%s?group=%s", baseURL, group)

	doc, _ := goquery.NewDocument(url)
	l := doc.Find("ul.pagination li").Length()

	if l == 0 {
		// there is no "ul.pagination li" element for some group only have 1 page
		return 1
	}
	// -2 is to deduct "上一頁" and "下一頁" element
	return l - 2
}

func getArticleURLs(group string) <-chan string {
	nPage := getNumberOfPage(group)
	articleURLs := make(chan string)

	var wg sync.WaitGroup
	wg.Add(nPage)

	baseURL := "https://ithelp.ithome.com.tw/ironman/signup/list"
	for i := 1; i <= nPage; i++ {
		go func(i int) {
			defer wg.Done()

			url := fmt.Sprintf("%s?group=%s&page=%d", baseURL, group, i)
			doc, _ := goquery.NewDocument(url)
			titleSelector := ".contestants-wrapper .contestants-list"

			doc.Find(titleSelector).Each(func(i int, s *goquery.Selection) {
				// filter out failed team
				isFailed := s.Find(".team-dashboard__box").HasClass("team-progress--fail")
				if !isFailed {
					// get article URL
					href, _ := s.Find("a.contestants-list__title").Attr("href")
					articleURLs <- href
				}
			})
		}(i)
	}

	go func() {
		// close channel when complete
		wg.Wait()
		close(articleURLs)
	}()

	return articleURLs
}

type article struct {
	title      string
	url        string
	subscriber int
}

func getArticles(urls <-chan string) []article {
	articles := []article{}

	var m sync.Mutex
	var wg sync.WaitGroup

	for url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			doc, _ := goquery.NewDocument(url)

			title := doc.Find(".qa-list__title.qa-list__title--ironman").Text()
			title = strings.TrimRight(strings.TrimSpace(title), " 系列")
			amount, _ := strconv.Atoi(doc.Find("span.subscription-amount").Text())

			info := article{
				title:      title,
				url:        url,
				subscriber: amount,
			}

			m.Lock()
			articles = append(articles, info)
			m.Unlock()
		}(url)
	}

	wg.Wait()

	return articles
}

func print(articles []article, group string) {
	data := [][]string{}
	for _, info := range articles {
		title := info.title
		limit := 119
		if len(title) > 119 {
			// break to next line if title is too long
			title = title[:limit] + "\n" + title[limit:]
		}
		if info.subscriber >= 10 {
			// only print articles whose number af amount >= 10
			subscriber := strconv.Itoa(info.subscriber)
			data = append(data, []string{"\n" + tablewriter.Pad(subscriber, " ", 4), title + "\n\n" + info.url})
		}
	}

	// set table format
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"訂閱數", "主題（" + group + "）"})
	table.SetRowLine(true)
	table.SetAutoWrapText(false)

	table.AppendBulk(data)
	table.Render()
}

func main() {
	groups := []string{"web", "software-dev", "self"}

	for _, group := range groups {
		urls := getArticleURLs(group)
		articles := getArticles(urls)

		// sort by number of subscriber
		sort.SliceStable(articles, func(i, j int) bool {
			return articles[i].subscriber > articles[j].subscriber
		})

		print(articles, group)
	}
}
