package download

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/davexpro/m3u8-dl/internal/m3u8"
	"github.com/davexpro/m3u8-dl/util"
)

const (
	tsExt            = ".ts"
	tsTempFileSuffix = "_tmp"
	progressWidth    = 40
)

type Downloader struct {
	queue    []int
	folder   string
	tsFolder string
	finish   int32
	segLen   int

	ffmpeg   bool
	threads  int
	filename string

	result *m3u8.Result
	sync.RWMutex
}

// NewDownloader .
func NewDownloader(url, output, filename string, threads int, ffmpeg bool) (*Downloader, error) {
	result, err := m3u8.FromURL(url)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(output, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create storage output failed: %s", err.Error())
	}
	tsFolder := filepath.Join(output, fmt.Sprintf("ts_%s", util.MD5Short(url)))
	if err := os.MkdirAll(tsFolder, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create ts output '[%s]' failed: %s", tsFolder, err.Error())
	}
	d := &Downloader{
		folder:   output,
		tsFolder: tsFolder,
		result:   result,
		filename: filename,
		threads:  threads,
		ffmpeg:   ffmpeg,
	}
	d.segLen = len(result.M3u8.Segments)
	d.queue = genSlice(d.segLen)
	return d, nil
}

// Start runs downloader
func (d *Downloader) Start() error {
	var wg sync.WaitGroup
	for i := 0; i < d.threads; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for {
				tsIdx, end, err := d.next()
				if end {
					break
				}
				if err != nil {
					continue
				}
				if err := d.download(tsIdx); err != nil {
					// Back into the queue, retry request
					fmt.Printf("[failed] %s\n", err.Error())
					if err := d.back(tsIdx); err != nil {
						fmt.Printf(err.Error())
					}
				}
			}
		}(i)
	}
	wg.Wait()

	if err := d.merge(); err != nil {
		return err
	}
	return nil
}

func (d *Downloader) download(segIndex int) error {
	tsName := tsFilename(segIndex)
	tsUrl := d.tsURL(segIndex)
	b, e := util.Get(tsUrl)
	if e != nil {
		return fmt.Errorf("request %s, %s", tsUrl, e.Error())
	}
	//noinspection GoUnhandledErrorResult
	defer b.Close()
	fPath := filepath.Join(d.tsFolder, tsName)
	fTemp := fPath + tsTempFileSuffix
	f, err := os.Create(fTemp)
	if err != nil {
		return fmt.Errorf("create file: %s, %s", tsName, err.Error())
	}
	bytes, err := ioutil.ReadAll(b)
	if err != nil {
		return fmt.Errorf("read bytes: %s, %s", tsUrl, err.Error())
	}
	sf := d.result.M3u8.Segments[segIndex]
	if sf == nil {
		return fmt.Errorf("invalid segment index: %d", segIndex)
	}
	key, ok := d.result.Keys[sf.KeyIndex]
	if ok && key != "" {
		bytes, err = util.AES128Decrypt(bytes, []byte(key),
			[]byte(d.result.M3u8.Keys[sf.KeyIndex].IV))
		if err != nil {
			return fmt.Errorf("decryt: %s, %s", tsUrl, err.Error())
		}
	}
	// https://en.wikipedia.org/wiki/MPEG_transport_stream
	// Some TS files do not start with SyncByte 0x47, they can not be played after merging,
	// Need to remove the bytes before the SyncByte 0x47(71).
	syncByte := uint8(71) //0x47
	bLen := len(bytes)
	for j := 0; j < bLen; j++ {
		if bytes[j] == syncByte {
			bytes = bytes[j:]
			break
		}
	}
	w := bufio.NewWriter(f)
	if _, err := w.Write(bytes); err != nil {
		return fmt.Errorf("write to %s: %s", fTemp, err.Error())
	}
	// Release file resource to rename file
	_ = f.Close()
	if err = os.Rename(fTemp, fPath); err != nil {
		return err
	}
	// Maybe it will be safer in this way...
	atomic.AddInt32(&d.finish, 1)
	//tool.DrawProgressBar("Downloading", float32(d.finish)/float32(d.segLen), progressWidth)
	fmt.Printf("[download %6.2f%%] %s\n", float32(d.finish)/float32(d.segLen)*100, tsUrl)
	return nil
}

func (d *Downloader) next() (segIndex int, end bool, err error) {
	d.Lock()
	defer d.Unlock()
	if len(d.queue) == 0 {
		err = fmt.Errorf("queue empty")
		if d.finish == int32(d.segLen) {
			end = true
			return
		}
		// Some segment indexes are still running.
		end = false
		return
	}
	segIndex = d.queue[0]
	d.queue = d.queue[1:]
	return
}

func (d *Downloader) back(segIndex int) error {
	d.Lock()
	defer d.Unlock()
	if sf := d.result.M3u8.Segments[segIndex]; sf == nil {
		return fmt.Errorf("invalid segment index: %d", segIndex)
	}
	d.queue = append(d.queue, segIndex)
	return nil
}

func (d *Downloader) merge() error {
	// In fact, the number of downloaded segments should be equal to number of m3u8 segments
	missingCount := 0
	mergeFile, err := os.Create(filepath.Join(d.tsFolder, "merge.txt"))
	for idx := 0; idx < d.segLen; idx++ {
		tsName := tsFilename(idx)
		f := filepath.Join(d.tsFolder, tsName)
		if _, err := os.Stat(f); err != nil {
			missingCount++
		} else {
			mergeFile.WriteString(fmt.Sprintf("file '%s'\n", tsName))
		}
	}
	_ = mergeFile.Close()

	if missingCount > 0 {
		fmt.Printf("[warning] %d files missing\n", missingCount)
	}

	if d.ffmpeg {
		if err := d.mergeFfmpeg(); err == nil {
			_ = os.RemoveAll(d.tsFolder)
			return nil
		}
	}

	// Create a TS file for merging, all segment files will be written to this file.
	mFilePath := filepath.Join(d.folder, d.filename)
	mFile, err := os.Create(mFilePath)
	if err != nil {
		return fmt.Errorf("create main TS file failed：%s", err.Error())
	}
	//noinspection GoUnhandledErrorResult
	defer mFile.Close()

	writer := bufio.NewWriter(mFile)
	mergedCount := 0
	for segIndex := 0; segIndex < d.segLen; segIndex++ {
		tsName := tsFilename(segIndex)
		bytes, err := ioutil.ReadFile(filepath.Join(d.tsFolder, tsName))
		_, err = writer.Write(bytes)
		if err != nil {
			continue
		}
		mergedCount++
		util.DrawProgressBar("merge",
			float32(mergedCount)/float32(d.segLen), progressWidth)
	}
	_ = writer.Flush()
	// Remove `ts` folder
	_ = os.RemoveAll(d.tsFolder)

	if mergedCount != d.segLen {
		fmt.Printf("[warning] \n%d files merge failed", d.segLen-mergedCount)
	}

	fmt.Printf("\n[output] %s\n", mFilePath)

	return nil
}

func (d *Downloader) mergeFfmpeg() error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return err
	}

	fmt.Println("[*] found `ffmpeg`", ffmpeg)

	// ffmpeg -f concat -safe 0 -i /path/to/merge.txt -c copy /path/to/merge.mp4
	cmd := exec.Command(ffmpeg, "-f", "concat", "-i", filepath.Join(d.tsFolder, "merge.txt"), "-c", "copy", filepath.Join(d.folder, d.filename))
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (d *Downloader) tsURL(segIndex int) string {
	seg := d.result.M3u8.Segments[segIndex]
	return util.ResolveURL(d.result.URL, seg.URI)
}

func tsFilename(ts int) string {
	return strconv.Itoa(ts) + tsExt
}

func genSlice(len int) []int {
	s := make([]int, 0)
	for i := 0; i < len; i++ {
		s = append(s, i)
	}
	return s
}
