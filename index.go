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

	var issuesDocuments []*IssueDocument

	issuesTotal := 0
	for page := 1; issuesTotal <= issuesCount; page++ {
		issues, err := client.ListIssues(ctx, page, MaxPerPage)
		if err != nil {
			return nil, err
		}

		for _, issue := range issues {
			issueDocument, err := fetchIssueDocument(ctx, client, issue)
			if err != nil {
				return nil, err
			}
			issuesDocuments = append(issuesDocuments, issueDocument)
		}
		issuesTotal += len(issues)
		log.Printf("[DEBUG] [%d/%d] Issues page indexed\n", issuesTotal, issuesCount)
	}
	return issuesDocuments, nil
}

func indexIssues(ctx context.Context, client *YandexTrackerClient, index bleve.Index) {
	issuesDocuments, err := fetchIssuesDocuments(ctx, client)
	if err != nil {
		log.Fatal(err)
	}

	for i, issue := range issuesDocuments {
		log.Printf("[DEBUG] [%d/%d] Indexing issue %s\n", i, len(issuesDocuments), issue.Key)
		if err := index.Index(issue.Key, issue); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	token := os.Getenv("YT_TOKEN")
	orgID := os.Getenv("YT_ORG_ID")
	indexPath := os.Getenv("BLEVE_INDEX_PATH")
	if token == "" || orgID == "" || indexPath == "" {
		log.Fatal("YT_TOKEN, YT_ORG_ID and BLEVE_INDEX_PATH must be set")
	}

	os.RemoveAll(indexPath)
	mapping := bleve.NewIndexMapping()
	index, err := bleve.New(indexPath, mapping)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	client := NewClient(token, orgID)

	indexIssues(ctx, &client, index)

	// index, err := bleve.Open(indexPath)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// query := bleve.NewMatchQuery("MSSQL")
	// req := bleve.NewSearchRequest(query)
	// req.Fields = []string{"*"}
	// req.Highlight = bleve.NewHighlight()

	// res, err := index.Search(req)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Printf("Successful: %d Total: %d Took: %f sec\n", res.Status.Successful, res.Total, res.Took.Seconds())

	// for i, hit := range res.Hits {
	// 	fmt.Printf("%d. ID: %s, Score: %f\n", i+1, hit.ID, hit.Score)
	// 	// fmt.Printf("   Fields: %v\n", hit.Fields) // All stored fields
	// 	fmt.Printf("   Fragments (Highlights):\n")
	// 	for field, fragments := range hit.Fragments {
	// 		fmt.Printf("     %s:\n", field)
	// 		for _, fragment := range fragments {
	// 			fmt.Printf("       - %s\n", fragment)
	// 		}
	// 	}
	// }
}
