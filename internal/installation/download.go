package installation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxToolArchive int64 = 256 << 20

// A partial artifact is only a cache entry, never an executable. A resumed body
// must start exactly at the saved offset; the complete locked digest is checked
// before extraction. Servers that ignore Range may restart the download safely.
func downloadTool(ctx context.Context, client *http.Client, dir string, t Tool) ([]byte, error) {
	if e := privateDir(dir); e != nil {
		return nil, e
	}
	path := filepath.Join(dir, t.SHA256+".part")
	for attempt := 0; attempt < 3; attempt++ {
		data, e := downloadAttempt(ctx, client, path, t)
		if e == nil {
			return data, nil
		}
		var network net.Error
		if ctx.Err() != nil || (!errors.As(e, &network) && !errors.Is(e, io.ErrUnexpectedEOF)) || attempt == 2 {
			return nil, e
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, errors.New("download retry budget exhausted")
}
func downloadAttempt(ctx context.Context, client *http.Client, path string, t Tool) ([]byte, error) {
	offset := int64(0)
	if st, e := os.Lstat(path); e == nil {
		if !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 || st.Size() > maxToolArchive {
			return nil, errors.New("unsafe partial artifact cache")
		}
		offset = st.Size()
		b, e := privateRead(path)
		if e != nil {
			return nil, e
		}
		if Digest(b) == t.SHA256 {
			return b, nil
		}
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Accept-Encoding", "identity")
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	res, e := client.Do(req)
	if e != nil {
		return nil, fmt.Errorf("download %s: %w", t.Name, e)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		offset = 0
	case http.StatusPartialContent:
		if !validRange(res.Header.Get("Content-Range"), offset, res.ContentLength) {
			return nil, errors.New("server returned a mismatched artifact range")
		}
	default:
		return nil, fmt.Errorf("download %s: HTTP %d", t.Name, res.StatusCode)
	}
	if res.ContentLength > maxToolArchive-offset {
		return nil, errors.New("artifact exceeds size limit")
	}
	flags := os.O_WRONLY | os.O_CREATE
	if offset == 0 {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}
	f, e := os.OpenFile(path, flags, 0600)
	if e != nil {
		return nil, e
	}
	_, copyErr := io.Copy(f, io.LimitReader(res.Body, maxToolArchive-offset+1))
	syncErr := f.Sync()
	closeErr := f.Close()
	if copyErr != nil {
		return nil, copyErr
	}
	if syncErr != nil {
		return nil, syncErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	b, e := privateRead(path)
	if e != nil {
		return nil, e
	}
	if int64(len(b)) > maxToolArchive || Digest(b) != t.SHA256 {
		_ = os.Remove(path)
		return nil, errors.New("download digest mismatch: " + t.Name)
	}
	return b, nil
}
func validRange(value string, offset, length int64) bool {
	if !strings.HasPrefix(value, "bytes ") {
		return false
	}
	span, total, ok := strings.Cut(strings.TrimPrefix(value, "bytes "), "/")
	if !ok {
		return false
	}
	start, end, ok := strings.Cut(span, "-")
	if !ok {
		return false
	}
	a, e1 := strconv.ParseInt(start, 10, 64)
	b, e2 := strconv.ParseInt(end, 10, 64)
	n, e3 := strconv.ParseInt(total, 10, 64)
	return e1 == nil && e2 == nil && e3 == nil && a == offset && b >= a && b == n-1 && n <= maxToolArchive && (length < 0 || length == b-a+1)
}
