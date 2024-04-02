package main

import (
	"crypto/md5"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/widget"
	"github.com/flopp/go-findfont"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	downLoadUrl = "http://自定义地址?name=%s&time=%d&sign=%s"
	CDNKey      = "自定义"
	pkgName     = "自定义"
	imageUrl    = "http://自定义地址"
)

type progressWriter struct {
	totalSize      int64
	downloadedSize int64
	startTime      time.Time
	progress       *widget.ProgressBar
	speed          *widget.Label
	remaining      *widget.Label
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.downloadedSize += int64(n)

	pw.progress.SetValue(float64(pw.downloadedSize) / float64(pw.totalSize))

	elapsedTime := time.Since(pw.startTime).Seconds()
	downloadSpeed := float64(pw.downloadedSize) / elapsedTime
	pw.speed.SetText(fmt.Sprintf("下载速度: %.2f KB/s", downloadSpeed/1024))

	remainingTime := (float64(pw.totalSize) - float64(pw.downloadedSize)) / downloadSpeed
	pw.remaining.SetText(fmt.Sprintf("剩余时间: %.2f s", remainingTime))

	return n, nil
}

func downloadFile(filepath string, url string, progress *widget.ProgressBar, speed *widget.Label, remaining *widget.Label) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	totalSize := resp.ContentLength
	pw := &progressWriter{
		totalSize: totalSize,
		startTime: time.Now(),
		progress:  progress,
		speed:     speed,
		remaining: remaining,
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, pw))
	if err != nil {
		return err
	}

	return nil
}

func init() {
	//设置中文字体:解决中文乱码问题
	fontPaths := findfont.List()
	for _, path := range fontPaths {
		if strings.Contains(path, "宋体.ttf") || strings.Contains(path, "新宋体.ttf") || strings.Contains(path, "微软雅黑.ttc") || strings.Contains(path, "simkai.ttf") {
			os.Setenv("FYNE_FONT", path)
			break
		}
	}
}

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func loadImageFromURL(url string) (fyne.Resource, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return fyne.NewStaticResource("imageFromURL", bytes), nil
}

type Url struct {
	Errno  uint32  `json:"errno"`
	Errstr string  `json:"errstr"`
	Info   UrlInfo `json:"info"`
}
type UrlInfo struct {
	CfgDownUrl  string `json:"cfg_down_url"`
	CfgDownName string `json:"cfg_down_name"`
}

func loadUrlFromCDN() (string, string, error) {
	name := pkgName
	time := time.Now().Unix()
	key := CDNKey
	sign := getMD5Hash(fmt.Sprintf("%s%d%s", name, time, key))
	url := fmt.Sprintf(downLoadUrl, name, time, sign)

	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	fmt.Println(string(bytes))
	var urlData Url
	err = json.Unmarshal(bytes, &urlData)
	if err != nil {
		return "", "", err
	}
	return urlData.Info.CfgDownUrl, urlData.Info.CfgDownName, nil
}

//go:embed icon.png
var iconData []byte

func main() {
	os.Setenv("FYNE_RENDERERS", "software")
	a := app.New()
	a.SetIcon(&fyne.StaticResource{
		StaticName:    "icon.png",
		StaticContent: iconData,
	})
	w := a.NewWindow("下载器")
	imageResource, err := loadImageFromURL(imageUrl)
	if err != nil {
		log.Fatal(err)
	}
	image := canvas.NewImageFromResource(imageResource)
	image.FillMode = canvas.ImageFillOriginal
	url, name, _ := loadUrlFromCDN()
	progress := widget.NewProgressBar()
	speed := widget.NewLabel("下载速度: 0 KB/s")
	remaining := widget.NewLabel("剩余时间: 0 s")
	defaultFolderPath := fmt.Sprintf("C:/1003B/%s", name)
	filepathLabel := widget.NewLabel(defaultFolderPath)
	selectFolderButton := widget.NewButton("选择下载目录", func() {
		dialog.ShowFolderOpen(func(dir fyne.ListableURI, err error) {
			if err == nil && dir != nil {
				// Add a default filename to the selected directory
				filepathLabel.SetText(fmt.Sprintf("%s/%s", dir.Name(), name))
			}
		}, w)
	})
	downloadButton := widget.NewButton("下载", func() {
		filepath := filepathLabel.Text
		// 检查目录是否存在，如果不存在则创建
		dir := filepath[:strings.LastIndex(filepath, "/")]
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.MkdirAll(dir, 0755)
		}
		err := downloadFile(filepath, url, progress, speed, remaining)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			// 执行下载的文件
			exec.Command(filepath).Start()
		}
	})

	content := container.NewVBox(
		image,
		selectFolderButton,
		filepathLabel,
		downloadButton,
		progress,
		speed,
		remaining,
	)

	w.SetContent(content)
	w.ShowAndRun()
}
