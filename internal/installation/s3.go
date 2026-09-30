package installation

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func mac(key []byte, s string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(s))
	return h.Sum(nil)
}
func signingKey(secret, day string) []byte {
	return mac(mac(mac(mac([]byte("AWS4"+secret), day), "us-east-1"), "s3"), "aws4_request")
}

// Fixed S3 SigV4 dialect for the installation's loopback endpoint. No SDK,
// credential provider chain, environment fallback or external account access.
func signed(req *http.Request, body []byte, c Credentials, now time.Time, presign bool) {
	day := now.UTC().Format("20060102")
	date := now.UTC().Format("20060102T150405Z")
	scope := day + "/us-east-1/s3/aws4_request"
	headers := "host;x-amz-content-sha256;x-amz-date"
	payload := Digest(body)
	canonicalHeaders := "host:" + req.URL.Host + "\nx-amz-content-sha256:" + payload + "\nx-amz-date:" + date + "\n"
	if presign {
		q := req.URL.Query()
		q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
		q.Set("X-Amz-Credential", c.AccessKey+"/"+scope)
		q.Set("X-Amz-Date", date)
		q.Set("X-Amz-Expires", "60")
		q.Set("X-Amz-SignedHeaders", "host")
		req.URL.RawQuery = strings.ReplaceAll(q.Encode(), "+", "%20")
		headers = "host"
		payload = "UNSIGNED-PAYLOAD"
		canonicalHeaders = "host:" + req.URL.Host + "\n"
	} else {
		req.Header.Set("X-Amz-Date", date)
		req.Header.Set("X-Amz-Content-Sha256", payload)
	}
	canonical := strings.Join([]string{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, canonicalHeaders, headers, payload}, "\n")
	signature := hex.EncodeToString(mac(signingKey(c.SecretKey, day), "AWS4-HMAC-SHA256\n"+date+"\n"+scope+"\n"+Digest([]byte(canonical))))
	if presign {
		req.URL.RawQuery += "&X-Amz-Signature=" + signature
	} else {
		req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+c.AccessKey+"/"+scope+", SignedHeaders="+headers+", Signature="+signature)
	}
}
func s3request(ctx context.Context, client *http.Client, method, address string, body []byte, c *Credentials, presign bool) ([]byte, int, http.Header, error) {
	req, e := http.NewRequestWithContext(ctx, method, address, bytes.NewReader(body))
	if e != nil {
		return nil, 0, nil, e
	}
	if c != nil {
		signed(req, body, *c, time.Now(), presign)
	}
	res, e := client.Do(req)
	if e != nil {
		return nil, 0, nil, e
	}
	defer res.Body.Close()
	b, e := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	return b, res.StatusCode, res.Header, e
}
func (w *Workflow) verifyS3(ctx context.Context, c Credentials, functional bool) error {
	address, stop, _, e := w.forward(ctx, "s3")
	if e != nil {
		return e
	}
	defer stop()
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	for _, test := range []struct {
		path string
		auth *Credentials
	}{{"/uploads?list-type=2", nil}, {"/atlas-denied?list-type=2", &c}} {
		_, status, _, e := s3request(ctx, client, "GET", address+test.path, nil, test.auth, false)
		if e != nil || status != 403 {
			return errors.New("S3 anonymous or cross-bucket access was not denied")
		}
	}
	b, status, _, e := s3request(ctx, client, "GET", address+"/uploads?list-type=2", nil, &c, false)
	if e != nil || status != 200 {
		return errors.New("S3 authorized uploads bucket unavailable")
	}
	var listing struct {
		XMLName xml.Name
		Name    string
	}
	if xml.Unmarshal(b, &listing) != nil || listing.XMLName.Local != "ListBucketResult" || listing.Name != "uploads" {
		return errors.New("S3 returned wrong bucket")
	}
	if !functional {
		return nil
	}
	key := "atlas-d1-" + w.Record.InstallID
	object := address + "/uploads/" + key
	body := []byte("Atlas D1 " + w.Record.InstallID + "\n")
	_, status, _, e = s3request(ctx, client, "PUT", object, body, &c, false)
	if e != nil || status != 200 {
		return errors.New("S3 object write failed")
	}
	// This fixed acceptance object contains no user data. Cleanup is bounded to
	// the per-install key even when a subsequent read/authorization check fails.
	objectDeleted := false
	defer func() {
		if !objectDeleted {
			_, _, _, _ = s3request(ctx, client, "DELETE", object, nil, &c, false)
		}
	}()
	for _, presign := range []bool{false, true} {
		got, status, _, e := s3request(ctx, client, "GET", object, nil, &c, presign)
		if e != nil || status != 200 || !bytes.Equal(got, body) {
			return errors.New("S3 signed/presigned object round trip failed")
		}
	}
	multipart := object + "-multipart"
	b, status, _, e = s3request(ctx, client, "POST", multipart+"?uploads=", nil, &c, false)
	if e != nil || status != 200 {
		return errors.New("S3 multipart initiation failed")
	}
	var upload struct {
		UploadID string `xml:"UploadId"`
	}
	if xml.Unmarshal(b, &upload) != nil || upload.UploadID == "" {
		return errors.New("invalid multipart upload identity")
	}
	query := "uploadId=" + url.QueryEscape(upload.UploadID)
	multipartComplete, multipartDeleted := false, false
	defer func() {
		if !multipartComplete {
			_, _, _, _ = s3request(ctx, client, "DELETE", multipart+"?"+query, nil, &c, false)
		}
		if !multipartDeleted {
			_, _, _, _ = s3request(ctx, client, "DELETE", multipart, nil, &c, false)
		}
	}()
	_, status, header, e := s3request(ctx, client, "PUT", multipart+"?partNumber=1&"+query, body, &c, false)
	if e != nil || status != 200 || header.Get("ETag") == "" {
		return errors.New("S3 multipart part failed")
	}
	completion, e := xml.Marshal(struct {
		XMLName xml.Name `xml:"CompleteMultipartUpload"`
		Part    struct {
			PartNumber int
			ETag       string
		}
	}{Part: struct {
		PartNumber int
		ETag       string
	}{1, header.Get("ETag")}})
	if e != nil {
		return e
	}
	b, status, _, e = s3request(ctx, client, "POST", multipart+"?"+query, completion, &c, false)
	if e != nil || status != 200 || !bytes.Contains(b, []byte("CompleteMultipartUploadResult")) {
		return errors.New("S3 multipart completion failed")
	}
	multipartComplete = true
	b, status, _, e = s3request(ctx, client, "GET", multipart, nil, &c, false)
	if e != nil || status != 200 || !bytes.Equal(b, body) {
		return errors.New("S3 multipart content differs")
	}
	_, status, _, e = s3request(ctx, client, "DELETE", object, nil, &c, false)
	if e != nil || status != 204 {
		return fmt.Errorf("S3 delete failed: status %d", status)
	}
	objectDeleted = true
	_, status, _, e = s3request(ctx, client, "GET", object, nil, &c, false)
	if e != nil || status != 404 {
		return errors.New("S3 deleted object remained readable")
	}
	_, status, _, e = s3request(ctx, client, "DELETE", multipart, nil, &c, false)
	if e != nil || status != 204 {
		return errors.New("S3 multipart object cleanup failed")
	}
	multipartDeleted = true
	return nil
}
