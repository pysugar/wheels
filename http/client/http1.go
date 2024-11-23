package client

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func (f *fetcher) doHTTP1(ctx context.Context, req *http.Request) (*http.Response, error) {
	f.printf("[%s] Falling back to HTTP/1.1", req.URL.RequestURI())

	conn, err := dialConn(req.Host, false)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	if req.Header == nil {
		req.Header = make(http.Header)
	}
	if req.Host == "" {
		req.Host = req.URL.Host
	}
	req.Header.Set("Connection", "close")

	return f.doHTTP1WithConn(ctx, req, conn)
}

func (f *fetcher) doHTTP1WithConn(ctx context.Context, req *http.Request, conn net.Conn) (*http.Response, error) {
	writer := bufio.NewWriter(conn)
	if err := req.Write(writer); err != nil {
		return nil, fmt.Errorf("failed to write HTTP/1.1 request: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush HTTP/1.1 request: %w", err)
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP/1.1 response: %w", err)
	}
	return resp, nil
}

func readHTTPResponse(reader *bufio.Reader, req *http.Request) (*http.Response, error) {
	statusLine, er := reader.ReadString('\n')
	if er != nil {
		return nil, er
	}
	statusLine = strings.TrimRight(statusLine, "\r\n")
	if statusLine == "" {
		return nil, errors.New("malformed HTTP response: empty status line")
	}

	parts := strings.SplitN(statusLine, " ", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("malformed HTTP response status line: %s", statusLine)
	}
	proto := parts[0]
	statusCodeStr := parts[1]
	statusText := ""
	if len(parts) > 2 {
		statusText = parts[2]
	}

	statusCode, er := strconv.Atoi(statusCodeStr)
	if er != nil {
		return nil, fmt.Errorf("invalid status code: %s", statusCodeStr)
	}

	respHeader := make(http.Header)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			return nil, fmt.Errorf("malformed header line: %s", line)
		}
		key := strings.TrimSpace(line[:colonIndex])
		value := strings.TrimSpace(line[colonIndex+1:])
		respHeader.Add(key, value)
	}

	var body io.ReadCloser = http.NoBody
	if req.Method != "HEAD" {
		if te := respHeader.Get("Transfer-Encoding"); strings.EqualFold(te, "chunked") {
			body = &chunkedReader{reader: reader}
		} else if cl := respHeader.Get("Content-Length"); cl != "" {
			contentLength, err := strconv.ParseInt(cl, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %s", cl)
			}
			body = io.NopCloser(io.LimitReader(reader, contentLength))
		} else {
			body = io.NopCloser(reader)
		}
	}

	resp := &http.Response{
		Status:        fmt.Sprintf("%d %s", statusCode, statusText),
		StatusCode:    statusCode,
		Proto:         proto,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        respHeader,
		Body:          body,
		ContentLength: -1, // 根据实际情况设置
		Request:       req,
	}

	return resp, nil
}

type chunkedReader struct {
	reader *bufio.Reader
	eof    bool
}

func (cr *chunkedReader) Read(p []byte) (int, error) {
	if cr.eof {
		return 0, io.EOF
	}

	line, err := cr.reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	line = strings.TrimSpace(line)
	size, err := strconv.ParseInt(line, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid chunk size: %s", line)
	}
	if size == 0 {
		cr.eof = true
		cr.reader.ReadString('\n')
		return 0, io.EOF
	}

	n, err := io.ReadFull(cr.reader, p[:int(size)])
	if err != nil {
		return n, err
	}

	cr.reader.ReadString('\n')
	return n, nil
}

func (cr *chunkedReader) Close() error {
	return nil
}
