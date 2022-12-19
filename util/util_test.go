package util

import (
	"fmt"
	"net/url"
	"testing"
)

func TestResolveURL(t *testing.T) {
	for i := 0; i < 1026; i++ {
		fmt.Println(fmt.Sprintf("file '/Users/dave/go/src/github.com/davexpro/m3u8-dl/down/ts_6e84bfcc2ca05f8d/%s.tx'"))
	}
	return

	testURL := "http://www.example.com/test/index.m3m8"
	u, err := url.Parse(testURL)
	if err != nil {
		t.Error(err)
	}

	result := ResolveURL(u, "videos/111111.ts")
	expected := "http://www.example.com/test/videos/111111.ts"
	if result != expected {
		t.Fatalf("wrong URL, expected: %s, result: %s", expected, result)
	}

	result = ResolveURL(u, "/videos/2222222.ts")
	expected = "http://www.example.com/videos/2222222.ts"
	if result != expected {
		t.Fatalf("wrong URL, expected: %s, result: %s", expected, result)
	}

	result = ResolveURL(u, "https://test.com/11111.key")
	expected = "https://test.com/11111.key"
	if result != expected {
		t.Fatalf("wrong URL, expected: %s, result: %s", expected, result)
	}
}
