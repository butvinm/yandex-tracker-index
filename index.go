package main

import (
	"context"
	"log"
	"os"
)

func main() {
	token := os.Getenv("YT_TOKEN")
	orgID := os.Getenv("YT_ORG_ID")
	if token == "" || orgID == "" {
		log.Fatal("YT_TOKEN and YT_ORG_ID must be set")
	}
	// indexPath := os.Getenv("BLEVE_INDEX_PATH")

	ctx := context.Background()

	client := NewClient(token, orgID)
	issuesCount, err := client.GetIssuesCount(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Issues count: %d\n", issuesCount)

	perPage := 50
	for page := 1; page*perPage <= 50; page++ {
		issues, err := client.ListIssues(ctx, page, 50)
		if err != nil {
			log.Fatal(err)
		}
		for _, issue := range issues {
			log.Println("Indexing ", issue.Key, len(issue.Attachments))
		}
	}

	// create := true
	// if create {
	// 	mapping := bleve.NewIndexMapping()

	// 	issueMapping := bleve.NewDocumentMapping()
	// 	// issueMapping.DefaultAnalyzer

	// 	mapping.AddDocumentMapping("issue", issueMapping)

	// 	os.RemoveAll(indexPath)
	// 	index, err := bleve.New(indexPath, mapping)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	client := NewClient(token, orgID)

	// 	issuesCount, err := client.GetIssuesCount()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	log.Printf("Issues count: %d\n", issuesCount)

	// 	page := 1
	// 	perPage := 50
	// 	for perPage*page < issuesCount {
	// 		issues, err := client.ListIssues(page, 50)
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		page += 1000
	// 		for _, issue := range issues {
	// 			log.Println("Indexing ", issue.ID)
	// 			index.Index(issue.Key, issue)
	// 		}
	// 	}
	// } else {
	// 	index, err := bleve.Open(indexPath)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	count, _ := index.DocCount()
	// 	log.Println("Doc count: ", count)

	// 	fields, _ := index.Fields()
	// 	log.Println("fields: ", fields)

	// 	// query := bleve.NewMatchAllQuery()
	// 	query := bleve.NewQueryStringQuery("key:ADCDQF")
	// 	searchRequest := bleve.NewSearchRequestOptions(query, 50, 0, true)

	// 	searchResult, err := index.Search(searchRequest)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	log.Println(searchResult.Status)
	// 	for _, doc := range searchResult.Hits {
	// 		log.Println(doc.ID)
	// 	}
	// }

	// query := bleve.NewMatchAllQuery()
	// search := bleve.NewSearchRequest(query)
	// result, err := index.Search(search)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// resultJSON, err := json.Marshal(result)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("Documents found:\n%s", resultJSON)
}
