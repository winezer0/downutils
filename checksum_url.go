package downutils

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

const maxChecksumResponseBytes = 1 << 20

func isChecksumURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func resolveChecksum(ctx context.Context, client *http.Client, checksumSpec, downloadURL string) (ParsedChecksum, error) {
	checksum := parseChecksum(checksumSpec)
	if !checksum.Enable || checksum.URL == "" {
		return checksum, nil
	}

	parsedURL, err := url.Parse(checksum.URL)
	if err != nil {
		return ParsedChecksum{}, checksumSourceError(fmt.Sprintf("invalid URL %q", checksum.URL))
	}
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)
	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return ParsedChecksum{}, checksumSourceError(fmt.Sprintf("invalid URL %q", checksum.URL))
	}
	response, err := HttpGet(ctx, client, parsedURL.String())
	if err != nil {
		return ParsedChecksum{}, checksumSourceFetchError(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxChecksumResponseBytes+1))
	if err != nil {
		return ParsedChecksum{}, checksumSourceFetchError(err)
	}
	if len(body) > maxChecksumResponseBytes {
		return ParsedChecksum{}, checksumSourceError("response exceeds 1 MiB")
	}

	resolved, err := parseRemoteChecksum(body, checksum.Type, downloadURL)
	if err != nil {
		return ParsedChecksum{}, checksumSourceError(err.Error())
	}
	return resolved, nil
}

func parseRemoteChecksum(body []byte, requiredType ChecksumType, downloadURL string) (ParsedChecksum, error) {
	content := strings.TrimSpace(strings.TrimPrefix(string(body), "\uFEFF"))
	if checksumType, ok := checksumTypeForValue(content); ok {
		if requiredType != "" && requiredType != checksumType {
			return ParsedChecksum{}, fmt.Errorf("checksum algorithm is %s, expected %s", checksumType, requiredType)
		}
		return ParsedChecksum{Type: checksumType, Value: strings.ToLower(content), Enable: true}, nil
	}

	wantedName, err := checksumURLFileName(downloadURL)
	if err != nil {
		return ParsedChecksum{}, err
	}
	var match *ParsedChecksum
	for _, line := range strings.Split(content, "\n") {
		checksum, fileName, ok := parseChecksumManifestLine(line)
		if !ok || checksumFileName(fileName) != wantedName {
			continue
		}
		if requiredType != "" && checksum.Type != requiredType {
			continue
		}
		if match != nil {
			return ParsedChecksum{}, fmt.Errorf("checksum manifest has multiple matching entries for %q", wantedName)
		}
		match = &checksum
	}
	if match == nil {
		return ParsedChecksum{}, fmt.Errorf("checksum manifest has no matching entry for %q", wantedName)
	}
	return *match, nil
}

func parseChecksumManifestLine(line string) (ParsedChecksum, string, bool) {
	line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
	if line == "" || strings.HasPrefix(line, "#") {
		return ParsedChecksum{}, "", false
	}
	separator := strings.IndexAny(line, " \t")
	if separator <= 0 {
		return ParsedChecksum{}, "", false
	}
	digest := line[:separator]
	checksumType, ok := checksumTypeForValue(digest)
	if !ok {
		return ParsedChecksum{}, "", false
	}
	fileName := strings.TrimLeft(line[separator:], " \t")
	if strings.HasPrefix(fileName, "*") {
		fileName = fileName[1:]
	}
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return ParsedChecksum{}, "", false
	}
	return ParsedChecksum{Type: checksumType, Value: strings.ToLower(digest), Enable: true}, fileName, true
}

func checksumTypeForValue(value string) (ChecksumType, bool) {
	if _, err := hex.DecodeString(value); err != nil {
		return "", false
	}
	switch len(value) {
	case md5.Size * 2:
		return ChecksumMD5, true
	case sha1.Size * 2:
		return ChecksumSHA1, true
	case sha256.Size * 2:
		return ChecksumSHA256, true
	default:
		return "", false
	}
}

func checksumURLFileName(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse download URL: %w", err)
	}
	name := path.Base(parsed.Path)
	if parsed.Path == "" || name == "." || name == "/" {
		return "", errors.New("download URL has no file name for checksum lookup")
	}
	return name, nil
}

func checksumFileName(name string) string {
	return path.Base(strings.ReplaceAll(name, `\`, "/"))
}

func checksumSourceError(message string) error {
	return DownloadError{Message: "checksum source failed: " + message, Type: ErrChecksumSourceFailed}
}

func checksumSourceFetchError(err error) error {
	var downloadErr DownloadError
	if errors.As(err, &downloadErr) {
		if downloadErr.Type == ErrDownloadCancelled {
			return downloadErr
		}
		return DownloadError{
			StatusCode: downloadErr.StatusCode,
			Message:    "checksum source failed: " + downloadErr.Message,
			Type:       ErrChecksumSourceFailed,
		}
	}
	return checksumSourceError(err.Error())
}
