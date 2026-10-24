package storage

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// S3Config contains options for connecting to AWS S3, MinIO, or Cloudflare R2.
type S3Config struct {
	Endpoint        string // e.g. "https://s3.amazonaws.com" or "http://localhost:9000"
	Bucket          string
	Region          string // default: "us-east-1"
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool // set true for MinIO or local emulators
	HTTPClient      *http.Client
}

// S3Store implements Store targeting S3-compatible object storage backends.
type S3Store struct {
	endpoint     string
	bucket       string
	region       string
	accessKey    string
	secretKey    string
	usePathStyle bool
	client       *http.Client
}

// NewS3Store initializes an S3Store.
func NewS3Store(cfg S3Config) (*S3Store, error) {
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, fmt.Errorf("storage: s3 bucket name is required")
	}

	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" {
		endpoint = "https://s3.amazonaws.com"
	}
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "us-east-1"
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &S3Store{
		endpoint:     endpoint,
		bucket:       cfg.Bucket,
		region:       region,
		accessKey:    cfg.AccessKeyID,
		secretKey:    cfg.SecretAccessKey,
		usePathStyle: cfg.UsePathStyle,
		client:       client,
	}, nil
}

func (s *S3Store) objectURL(key string) string {
	clean := CleanKey(key)
	if s.usePathStyle || strings.Contains(s.endpoint, "localhost") || strings.Contains(s.endpoint, "127.0.0.1") {
		return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, clean)
	}
	// Virtual host style: https://bucket.s3.region.amazonaws.com/key
	parts := strings.SplitN(s.endpoint, "://", 2)
	scheme := parts[0]
	host := parts[1]
	return fmt.Sprintf("%s://%s.%s/%s", scheme, s.bucket, host, clean)
}

// Put uploads an object to S3.
func (s *S3Store) Put(key string, data []byte) error {
	url := s.objectURL(key)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("storage: build s3 put request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	s.signRequest(req, data)

	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("storage: s3 put %q failed: %w", key, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("storage: s3 put %q error (status %d): %s", key, res.StatusCode, string(body))
	}
	return nil
}

// Get downloads an object from S3.
func (s *S3Store) Get(key string) ([]byte, error) {
	url := s.objectURL(key)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("storage: build s3 get request: %w", err)
	}

	s.signRequest(req, nil)

	res, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("storage: s3 get %q failed: %w", key, err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("storage: s3 get %q error (status %d): %s", key, res.StatusCode, string(body))
	}

	return io.ReadAll(res.Body)
}

// Delete removes an object from S3.
func (s *S3Store) Delete(key string) error {
	url := s.objectURL(key)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("storage: build s3 delete request: %w", err)
	}

	s.signRequest(req, nil)

	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("storage: s3 delete %q failed: %w", key, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("storage: s3 delete %q error (status %d): %s", key, res.StatusCode, string(body))
	}
	return nil
}

// Stat retrieves object metadata via HEAD.
func (s *S3Store) Stat(key string) (ObjectInfo, error) {
	url := s.objectURL(key)
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("storage: build s3 head request: %w", err)
	}

	s.signRequest(req, nil)

	res, err := s.client.Do(req)
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("storage: s3 stat %q failed: %w", key, err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return ObjectInfo{}, ErrNotFound
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return ObjectInfo{}, fmt.Errorf("storage: s3 stat %q error (status %d)", key, res.StatusCode)
	}

	return ObjectInfo{
		Key:  filepath.ToSlash(CleanKey(key)),
		Size: res.ContentLength,
		Mode: fs.FileMode(0o644),
	}, nil
}

// signRequest applies AWS SigV4 authorization headers.
func (s *S3Store) signRequest(req *http.Request, payload []byte) {
	if s.accessKey == "" || s.secretKey == "" {
		return
	}

	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	dateStamp := t.Format("20060102")

	req.Header.Set("x-amz-date", amzDate)
	host := req.URL.Host
	req.Header.Set("Host", host)

	payloadHash := sha256Hex(payload)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	canonicalURI := req.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalReq := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method, canonicalURI, req.URL.RawQuery, canonicalHeaders, signedHeaders, payloadHash)

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, s.region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, sha256Hex([]byte(canonicalReq)))

	signingKey := getSignatureKey(s.secretKey, dateStamp, s.region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSignatureKey(key, dateStamp, regionName, serviceName string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+key), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(regionName))
	kService := hmacSHA256(kRegion, []byte(serviceName))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

var _ Store = (*S3Store)(nil)
