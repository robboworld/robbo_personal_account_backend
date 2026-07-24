package usersearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/spf13/viper"
)

const defaultIndex = "lk_lms_users"

// Hit is returned by Search for admin autocomplete.
type Hit struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
}

// Service searches LMS users via Elasticsearch with MySQL fallback.
type Service struct {
	es        *elasticsearch.Client
	index     string
	esReady   bool
	mu        sync.RWMutex
	stopCh    chan struct{}
	stoppedCh chan struct{}
}

// NewFromConfig builds a Service. ES is optional; MySQL fallback always works.
func NewFromConfig() *Service {
	s := &Service{
		index:     strings.TrimSpace(viper.GetString("elasticsearch.index")),
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
	if s.index == "" {
		s.index = defaultIndex
	}

	enabled := viper.GetBool("elasticsearch.enabled")
	url := strings.TrimSpace(viper.GetString("elasticsearch.url"))
	if !enabled || url == "" {
		log.Printf("usersearch: elasticsearch disabled (enabled=%v url=%q); MySQL fallback only", enabled, url)
		close(s.stoppedCh)
		return s
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{url},
	})
	if err != nil {
		log.Printf("usersearch: elasticsearch client: %v; MySQL fallback only", err)
		close(s.stoppedCh)
		return s
	}
	s.es = es

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.ping(ctx); err != nil {
		log.Printf("usersearch: elasticsearch unreachable at %s: %v; MySQL fallback only", url, err)
		close(s.stoppedCh)
		return s
	}

	s.setReady(true)
	if err := s.EnsureIndex(context.Background()); err != nil {
		log.Printf("usersearch: ensure index: %v", err)
	}
	if n, err := s.ReindexAll(context.Background()); err != nil {
		log.Printf("usersearch: initial reindex: %v", err)
	} else {
		log.Printf("usersearch: initial reindex ok, documents=%d", n)
	}

	intervalMin := viper.GetInt("elasticsearch.reindexIntervalMinutes")
	if intervalMin > 0 {
		go s.loopReindex(time.Duration(intervalMin) * time.Minute)
	} else {
		close(s.stoppedCh)
	}
	return s
}

func (s *Service) setReady(v bool) {
	s.mu.Lock()
	s.esReady = v
	s.mu.Unlock()
}

func (s *Service) ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.es != nil && s.esReady
}

func (s *Service) ping(ctx context.Context) error {
	res, err := s.es.Ping(s.es.Ping.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("ping status %s", res.Status())
	}
	return nil
}

func (s *Service) loopReindex(interval time.Duration) {
	defer close(s.stoppedCh)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-t.C:
			if !s.ready() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := s.ping(ctx)
				cancel()
				if err != nil {
					continue
				}
				s.setReady(true)
				_ = s.EnsureIndex(context.Background())
			}
			if n, err := s.ReindexAll(context.Background()); err != nil {
				log.Printf("usersearch: periodic reindex: %v", err)
			} else {
				log.Printf("usersearch: periodic reindex ok, documents=%d", n)
			}
		}
	}
}

// Stop stops the periodic reindex ticker.
func (s *Service) Stop() {
	select {
	case <-s.stoppedCh:
		return
	default:
	}
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
	<-s.stoppedCh
}

const indexMapping = `{
  "settings": {
    "analysis": {
      "analyzer": {
        "username_autocomplete": {
          "tokenizer": "username_edge",
          "filter": ["lowercase"]
        }
      },
      "tokenizer": {
        "username_edge": {
          "type": "edge_ngram",
          "min_gram": 1,
          "max_gram": 20,
          "token_chars": ["letter", "digit", "punctuation", "symbol"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "username": {
        "type": "text",
        "analyzer": "username_autocomplete",
        "search_analyzer": "standard",
        "fields": { "keyword": { "type": "keyword" } }
      },
      "email": {
        "type": "text",
        "analyzer": "username_autocomplete",
        "search_analyzer": "standard",
        "fields": { "keyword": { "type": "keyword" } }
      },
      "fullName": {
        "type": "text",
        "analyzer": "standard"
      },
      "isActive": { "type": "boolean" }
    }
  }
}`

// EnsureIndex creates the users index if missing.
func (s *Service) EnsureIndex(ctx context.Context) error {
	if s.es == nil {
		return fmt.Errorf("elasticsearch not configured")
	}
	res, err := s.es.Indices.Exists([]string{s.index}, s.es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		return nil
	}
	createRes, err := s.es.Indices.Create(
		s.index,
		s.es.Indices.Create.WithContext(ctx),
		s.es.Indices.Create.WithBody(strings.NewReader(indexMapping)),
	)
	if err != nil {
		return err
	}
	defer createRes.Body.Close()
	if createRes.IsError() {
		body, _ := io.ReadAll(createRes.Body)
		return fmt.Errorf("create index: %s %s", createRes.Status(), string(body))
	}
	return nil
}

type esDoc struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
	IsActive bool   `json:"isActive"`
}

// ReindexAll bulk-loads auth_user into Elasticsearch.
func (s *Service) ReindexAll(ctx context.Context) (int, error) {
	if s.es == nil {
		return 0, fmt.Errorf("elasticsearch not configured")
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	const page = 500
	offset := 0
	total := 0
	for {
		batch, err := reader.ListAuthUsersForIndex(page, offset)
		if err != nil {
			return total, err
		}
		if len(batch) == 0 {
			break
		}
		if err := s.bulkIndex(ctx, batch); err != nil {
			return total, err
		}
		total += len(batch)
		offset += len(batch)
		if len(batch) < page {
			break
		}
	}
	return total, nil
}

func (s *Service) bulkIndex(ctx context.Context, hits []lmsdb.AuthUserSearchHit) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, h := range hits {
		meta := map[string]map[string]string{
			"index": {"_index": s.index, "_id": strconv.FormatInt(h.ID, 10)},
		}
		if err := enc.Encode(meta); err != nil {
			return err
		}
		doc := esDoc{
			ID:       strconv.FormatInt(h.ID, 10),
			Username: h.Username,
			Email:    h.Email,
			FullName: strings.TrimSpace(h.FullName),
			IsActive: h.IsActive,
		}
		if err := enc.Encode(doc); err != nil {
			return err
		}
	}
	req := esapi.BulkRequest{
		Body:    bytes.NewReader(buf.Bytes()),
		Refresh: "false",
	}
	res, err := req.Do(ctx, s.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("bulk index: %s %s", res.Status(), string(body))
	}
	var parsed struct {
		Errors bool `json:"errors"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return err
	}
	if parsed.Errors {
		return fmt.Errorf("bulk index reported item errors")
	}
	return nil
}

// Search finds users by query string. Tries ES first, falls back to MySQL.
func (s *Service) Search(ctx context.Context, q string, limit int) ([]Hit, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []Hit{}, nil
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	if s.ready() {
		hits, err := s.searchES(ctx, q, limit)
		if err == nil {
			return hits, nil
		}
		log.Printf("usersearch: es search failed, falling back to mysql: %v", err)
	}
	return s.searchMySQL(q, limit)
}

func (s *Service) searchES(ctx context.Context, q string, limit int) ([]Hit, error) {
	body := map[string]interface{}{
		"size": limit,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{"term": map[string]interface{}{"isActive": true}},
				},
				"should": []map[string]interface{}{
					{"match": map[string]interface{}{"username": map[string]interface{}{"query": q, "operator": "and", "boost": 3}}},
					{"match": map[string]interface{}{"email": map[string]interface{}{"query": q, "operator": "and", "boost": 2}}},
					{"match": map[string]interface{}{"fullName": map[string]interface{}{"query": q, "operator": "and"}}},
					{"prefix": map[string]interface{}{"username.keyword": map[string]interface{}{"value": q, "case_insensitive": true, "boost": 4}}},
					{"prefix": map[string]interface{}{"email.keyword": map[string]interface{}{"value": q, "case_insensitive": true, "boost": 2}}},
				},
				"minimum_should_match": 1,
			},
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, err
	}
	res, err := s.es.Search(
		s.es.Search.WithContext(ctx),
		s.es.Search.WithIndex(s.index),
		s.es.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("search: %s %s", res.Status(), string(b))
	}
	var parsed struct {
		Hits struct {
			Hits []struct {
				Source esDoc `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	out := make([]Hit, 0, len(parsed.Hits.Hits))
	for _, h := range parsed.Hits.Hits {
		out = append(out, Hit{
			ID:       h.Source.ID,
			Username: h.Source.Username,
			Email:    h.Source.Email,
			FullName: h.Source.FullName,
		})
	}
	return out, nil
}

func (s *Service) searchMySQL(q string, limit int) ([]Hit, error) {
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	rows, err := reader.SearchAuthUsersPrefix(q, limit)
	if err != nil {
		return nil, err
	}
	out := make([]Hit, 0, len(rows))
	for _, r := range rows {
		out = append(out, Hit{
			ID:       strconv.FormatInt(r.ID, 10),
			Username: r.Username,
			Email:    r.Email,
			FullName: strings.TrimSpace(r.FullName),
		})
	}
	return out, nil
}

// ResolveUsernameToID looks up an active LMS user by username and returns string id.
func ResolveUsernameToID(username string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", fmt.Errorf("username required")
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	u, err := reader.LookupAuthUserByUsername(username)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", fmt.Errorf("user not found")
	}
	return strconv.FormatInt(u.ID, 10), nil
}
