package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var ACCESS_KEY = ""
var SECRET_KEY = ""
var ENDPOINT = ""
var BUCKET = ""
var DOCUMENT_ROOT = ""
var DURATION_HOUR = 24
var PORT = 8080

var s3client *minio.Client

func init() {
	initEnv()
	initMinio()
}

func initEnv() {
	if len(os.Args) > 1 {
		err := godotenv.Load(os.Args[1])
		if err != nil {
			log.Fatalf("Error loading %s file (from args)\n%v", os.Args[1], err)
		}
	} else {
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file\n%v", err)
		}
	}

	ACCESS_KEY = os.Getenv("ACCESS_KEY")
	SECRET_KEY = os.Getenv("SECRET_KEY")
	ENDPOINT = os.Getenv("ENDPOINT")
	BUCKET = os.Getenv("BUCKET")
	DOCUMENT_ROOT = os.Getenv("DOCUMENT_ROOT")
	DURATION_HOUR, _ = strconv.Atoi(os.Getenv("DURATION_HOUR"))
	PORT, _ = strconv.Atoi(os.Getenv("PORT"))
}

func initMinio() {
	_s3client, err := minio.New(ENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV2(ACCESS_KEY, SECRET_KEY, ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalln(err)
	}
	s3client = _s3client
}

func preSign(path string) (string, error) {
	if s3client == nil {
		panic("nil")
	}

	// Set request parameters
	reqParams := make(url.Values)

	// Gernerate presigned get object url.
	presignedURL, err := s3client.PresignedGetObject(
		context.Background(),
		BUCKET,
		path,
		time.Duration(DURATION_HOUR)*time.Hour,
		reqParams,
	)
	if err != nil {
		log.Fatalln(err)
	}

	return presignedURL.String(), nil
}

func processFile(path string) (string, error) {
	log.Printf("processFile: %s", path)

	if !strings.HasSuffix(path, ".html") {
		return "", errors.New("not supported type")
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		msg := fmt.Sprintf("open error: %s", path)
		return "", errors.New(msg)
	}

	return processContent(contentBytes)
}

func processContent(html []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return "", err
	}

	attrs := []string{
		"data-src",
		"src",
	}
	doc.Find(`img`).Each(func(_ int, s *goquery.Selection) {
		for _, attr := range attrs {
			value, exists := s.Attr(attr)
			if exists {
				value, err = preSign(value)
				if err == nil {
					s.SetAttr(attr, value)
				}
			}
		}
	})

	return doc.Html()
}

func main() {
	baseFullPath, err := filepath.Abs(DOCUMENT_ROOT)
	if err != nil {
		log.Fatalln(err)
	}
	baseFullPath = baseFullPath + "/"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Path[1:]
		if len(file) == 0 || strings.HasSuffix(file, "/") {
			file = file + "index.html"
		}

		joined := filepath.Join(DOCUMENT_ROOT, file)

		fullPath, err := filepath.Abs(joined)
		if err != nil {
			log.Println(err)
			http.NotFound(w, r)
			return
		}

		if !strings.HasPrefix(fullPath, baseFullPath) {
			log.Printf("out of docroot: fullPath=%s, baseFullPath=%s", fullPath, baseFullPath)
			http.NotFound(w, r)
			return
		}

		content, err := processFile(fullPath)
		if err != nil {
			log.Println(err)
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, content)
	})

	addr := fmt.Sprintf(":%d", PORT)
	log.Println(addr)
	http.ListenAndServe(addr, nil)
}
