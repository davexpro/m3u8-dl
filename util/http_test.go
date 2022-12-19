package util

import (
	"io/ioutil"
	"testing"
)

func TestGet(t *testing.T) {
	body, err := Get("https://raw.githubusercontent.com/DavexPro/m3u8-dl/main/LICENSE")
	if err != nil {
		t.Error(err)
	}
	defer body.Close()
	_, err = ioutil.ReadAll(body)
	if err != nil {
		t.Error(err)
	}
}
