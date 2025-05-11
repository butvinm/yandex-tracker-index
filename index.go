package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
)

var textLikeFiles = func() map[string]bool {
	exts := []string{
		"txt", "text", "log", "csv", "json", "xml", "css", "scss", "less",
		"js", "jsx", "ts", "tsx", "go", "py", "java", "c", "cpp", "h", "hpp", "rb",
		"php", "swift", "pl", "sh", "bash", "zsh", "yaml", "yml", "toml", "ini", "conf",
		"config", "env", "lock", "md", "markdown", "rst", "adoc", "asciidoc", "bat", "cmd",
		"ps1", "psm1", "r", "m", "mat", "sas", "sql", "vb", "vbs", "cs", "fs", "fsx",
		"dart", "kotlin", "scala", "groovy", "lua", "rust", "rs", "vue", "elm", "ex", "exs",
		"hs", "clj", "d", "jl", "nim", "svg", "graphql", "gql", "proto", "avro", "diff", "patch",
		"properties", "cfg", "htaccess", "gitignore", "dockerignore", "rtf", "sdoc",
	}

	res := make(map[string]bool, len(exts))
	for _, ext := range exts {
		res[strings.ToLower("."+ext)] = true
	}
	return res
}()

type AttachmentDocument struct {
	Attachment
	Content string
}

type CommentDocument struct {
	Comment
	Attachments []*AttachmentDocument
}

type IssueDocument struct {
	Issue
	Comments    []*CommentDocument
	Attachments []*AttachmentDocument
}

func fetchAttachmentDocument(ctx context.Context, client *YandexTrackerClient, issueKey string, attachmentInfo *AttachmentInfo) (*AttachmentDocument, error) {
	log.Printf("[DEBUG] Fetching attachment %s\n", attachmentInfo.Display)

	attachment, err := client.GetAttachment(ctx, issueKey, attachmentInfo.ID)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(attachment.Name)
	extLower := strings.ToLower(ext)
	if !textLikeFiles[extLower] {
		log.Printf("[DEBUG] Skip document with unsupported extension\n")
		return nil, nil
	}

	content, err := client.DownloadAttachment(ctx, issueKey, attachment.ID, attachment.Name)
	if err != nil {
		return nil, err
	}

	return &AttachmentDocument{
		Attachment: *attachment,
		Content:    string(content),
	}, nil
}

func fetchCommentDocument(ctx context.Context, client *YandexTrackerClient, issueKey string, comment *Comment) (*CommentDocument, error) {
	log.Printf("[DEBUG] Fetching comment %d\n", comment.ID)

	var attachmentsDocuments []*AttachmentDocument
	for _, attachment := range comment.Attachments {
		attachmentDocument, err := fetchAttachmentDocument(ctx, client, issueKey, &attachment)
		if err != nil {
			return nil, err
		}
		if attachmentDocument != nil {
			attachmentsDocuments = append(attachmentsDocuments, attachmentDocument)
		}
	}
	return &CommentDocument{
		Comment:     *comment,
		Attachments: attachmentsDocuments,
	}, nil
}

func fetchIssueDocument(ctx context.Context, client *YandexTrackerClient, issue *Issue) (*IssueDocument, error) {
	log.Printf("[DEBUG] Fetching issue %s\n", issue.Key)

	comments, err := client.ListIssueComments(ctx, issue.Key, 0, MaxPerPage)
	if err != nil {
		return nil, err
	}

	var commentsDocuments []*CommentDocument
	for _, comment := range comments {
		commentDocument, err := fetchCommentDocument(ctx, client, issue.Key, comment)
		if err != nil {
			return nil, err
		}
		commentsDocuments = append(commentsDocuments, commentDocument)
	}

	var attachmentsDocuments []*AttachmentDocument
	for _, attachment := range issue.Attachments {
		attachmentDocument, err := fetchAttachmentDocument(ctx, client, issue.Key, &attachment)
		if err != nil {
			return nil, err
		}
		if attachmentDocument != nil {
			attachmentsDocuments = append(attachmentsDocuments, attachmentDocument)
		}
	}

	return &IssueDocument{
		Issue:       *issue,
		Comments:    commentsDocuments,
		Attachments: attachmentsDocuments,
	}, nil
}

func fetchIssuesDocuments(ctx context.Context, client *YandexTrackerClient) ([]*IssueDocument, error) {
	issuesCount, err := client.GetIssuesCount(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Total issues: %d\n", issuesCount)

	pagesCount := (issuesCount + MaxPerPage - 1) / MaxPerPage

	type pageResult struct {
		issues []*Issue
		err    error
	}
	pageResultChan := make(chan pageResult, pagesCount)

	pageSemaphore := make(chan struct{}, 10)

	for page := 1; page <= pagesCount; page++ {
		go func(page int) {
			pageSemaphore <- struct{}{}
			defer func() { <-pageSemaphore }()

			log.Printf("[INFO] Fetching issues page %d/%d\n", page, pagesCount)
			pageIssues, err := client.ListIssues(ctx, page, MaxPerPage)
			pageResultChan <- pageResult{pageIssues, err}
		}(page)
	}

	var issues []*Issue
	for range pagesCount {
		res := <-pageResultChan
		if res.err != nil {
			return nil, res.err
		}
		issues = append(issues, res.issues...)
	}

	log.Printf("[INFO] Collected %d issues, now fetching details\n", len(issues))

	type docResult struct {
		doc *IssueDocument
		err error
	}
	docResultChan := make(chan docResult, len(issues))

	docSemaphore := make(chan struct{}, 10)

	for _, issue := range issues {
		go func(issue *Issue) {
			docSemaphore <- struct{}{}
			defer func() { <-docSemaphore }()

			doc, err := fetchIssueDocument(ctx, client, issue)
			docResultChan <- docResult{doc, err}
		}(issue)
	}

	var issuesDocuments []*IssueDocument
	for i := range len(issues) {
		res := <-docResultChan
		if res.err != nil {
			return nil, res.err
		}
		log.Printf("[INFO] [%d/%d] Fetched issue document\n", i+1, len(issues))
		issuesDocuments = append(issuesDocuments, res.doc)
	}

	return issuesDocuments, nil
}

func indexIssues(ctx context.Context, client *YandexTrackerClient, index bleve.Index) {
	issuesDocuments, err := fetchIssuesDocuments(ctx, client)
	if err != nil {
		log.Fatal(err)
	}

	for i, issue := range issuesDocuments {
		log.Printf("[INFO] [%d/%d] Indexing issue %s\n", i, len(issuesDocuments), issue.Key)
		if err := index.Index(issue.Key, issue); err != nil {
			log.Fatal(err)
		}
	}
}

func _() {
	token := os.Getenv("YT_TOKEN")
	orgID := os.Getenv("YT_ORG_ID")
	indexPath := os.Getenv("BLEVE_INDEX_PATH")
	if token == "" || orgID == "" || indexPath == "" {
		log.Fatal("YT_TOKEN, YT_ORG_ID and BLEVE_INDEX_PATH must be set")
	}

	mapping := bleve.NewIndexMapping()
	index, err := bleve.Open(indexPath)
	if err == nil {
		log.Printf("[INFO] Opening existing index at %s", indexPath)
	} else if err == bleve.ErrorIndexPathExists {
		log.Printf("[INFO] Creating a new index at %s", indexPath)
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal(err)
	}

	ctx := context.Background()
	client := NewClient(token, orgID)

	indexIssues(ctx, &client, index)
}
