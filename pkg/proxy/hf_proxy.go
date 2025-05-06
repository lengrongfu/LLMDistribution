package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/lengrongfu/LLMDistribution/pkg/utils"
)

type Proxy struct {
	// Add fields for proxy configuration
	baseURL       string
	client        *http.Client
	proxy         *httputil.ReverseProxy
	FallbackProxy bool
	baseDir       string
	bufferPool    sync.Pool
}

func NewProxy(baseURL string) *Proxy {
	if baseURL == "" {
		baseURL = "https://huggingface.co"
	}
	target, _ := url.Parse(baseURL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Error proxying to %s: %v", target, err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
	}
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	p := &Proxy{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		proxy: proxy,
	}
	p.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 64*1024)
		},
	}
	return p
}

func (p *Proxy) HandleGetModelIndex(w http.ResponseWriter, r *http.Request) {
	// if p.FallbackProxy {
	// 	resp, err := p.GetModelIndex(r)
	// 	if err != nil {
	// 		http.Error(w, fmt.Sprintf("failed to get model index: %v", err), http.StatusServiceUnavailable)
	// 		return
	// 	}
	// 	for key, values := range resp.Header {
	// 		for _, value := range values {
	// 			w.Header().Add(key, value)
	// 		}
	// 	}
	// 	defer resp.Body.Close()
	// 	vars := mux.Vars(r)
	// 	modelID := vars["model_id"]
	// 	modelPath := utils.ConvertModelIDToHFPath(modelID)
	// 	log.Printf("Downloading model index for %s", modelPath)
	// 	os.MkdirAll(filepath.Join(p.baseDir, "hub", modelPath), 0755)
	// 	modelIndexPath := filepath.Join(p.baseDir, "hub", modelPath, ".modeindex")
	// 	file, err := os.Create(modelIndexPath)
	// 	if err != nil {
	// 		http.Error(w, fmt.Sprintf("failed to create file: %v", err), http.StatusServiceUnavailable)
	// 		return
	// 	}
	// 	teeReader := io.TeeReader(resp.Body, file)
	// 	io.Copy(w, teeReader)
	// 	io.Copy(io.Discard, resp.Body)
	// 	return
	// }
	p.proxy.ServeHTTP(w, r)

}

func (p *Proxy) HandleGetModelFile(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
	// if p.FallbackProxy {
	// 	// Create a new Hugging Face client
	// 	client, err := api.NewApi()
	// 	if err != nil {
	// 		http.Error(w, fmt.Sprintf("failed to create Hugging Face client: %v", err), http.StatusServiceUnavailable)
	// 		return
	// 	}
	// 	vars := mux.Vars(r)
	// 	modelID := vars["model_id"]
	// 	filename := vars["filename"]
	// 	// Get the model
	// 	model := client.Model(modelID)
	// 	log.Printf("Downloading %s", filename)
	// 	// Use the Get method which will download the file to the HF_HOME cache directory
	// 	// and return the path to the cached file
	// 	_, err = model.Get(filename)
	// 	if err != nil {
	// 		http.Error(w, fmt.Sprintf("failed to download %s: %v", filename, err), http.StatusServiceUnavailable)
	// 		return
	// 	}
	// 	return
	// }
}

func (p *Proxy) WithFallbackProxy(fallback bool, baseDir string) {
	p.FallbackProxy = fallback
	p.baseDir = baseDir
	os.Setenv("HF_HOME", baseDir)
	log.Printf("Set HF_HOME environment variable to %s", baseDir)
}

func (p *Proxy) GetModelIndex(r *http.Request) (*http.Response, error) {
	// Create the URL to the Hugging Face API
	vars := mux.Vars(r)
	modelID := vars["model_id"]
	version := vars["version"]
	url := fmt.Sprintf("%s/api/models/%s/revision/%s", p.baseURL, modelID, version)
	// Create the context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// Send the request
	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	return resp, err
}

func (p *Proxy) WithModifyRequest(f func(*http.Response) error) {
	p.proxy.ModifyResponse = f
}
func (p *Proxy) WithModifyResponseToCache(resp *http.Response) error {
	if resp.Request.Method == "HEAD" {
		if location := resp.Header.Get("Location"); location != "" {
			go func() {
				f, err := p.CreateModelFile(resp, resp.Request)
				if err != nil {
					log.Printf("failed to create file: %+v", err)
					return
				}
				rsp, err := http.Get(location)
				if err != nil {
					log.Printf("failed to get file: %+v", err)
					return
				}
				defer rsp.Body.Close()
				defer f.Close()
				io.Copy(f, rsp.Body)
				log.Println("Write file done", f.Name())
			}()
			log.Println("HEAD request with Location header", location)
		}
		return nil
	}
	vars := mux.Vars(resp.Request)
	shaOrVersion := vars["sha"]
	var (
		f   *os.File
		err error
	)
	if shaOrVersion == "" {
		// save .modexlindex
		f, err = p.CreateModelIndexFile(resp.Request)
	} else {
		// other file
		f, err = p.CreateModelFile(resp, resp.Request)
	}
	if err != nil {
		log.Printf("failed to create file: %+v", err)
		return err
	}

	buf := p.bufferPool.Get().([]byte)
	defer p.bufferPool.Put(buf)
	resp.Body = io.NopCloser(io.TeeReader(
		resp.Body,
		&streamWriter{
			writer: f,
			buffer: buf,
		},
	))
	// if shaOrVersion != "" {
	// 	vars := mux.Vars(resp.Request)
	// 	modelID := vars["model_id"]
	// 	filename := vars["filename"]
	// 	commit, etag, err := getCommitAndEtag(resp)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	destDir := filepath.Join(p.path(modelID), "snapshots", commit)
	// 	if _, err := os.Stat(destDir); os.IsNotExist(err) {
	// 		os.MkdirAll(destDir, 0755)
	// 	}
	// 	destfile := filepath.Join(destDir, filename)
	// 	blobPath := filepath.Join(filepath.Join(p.path(modelID), "blobs"), etag)
	// 	symlinkOrRename(blobPath, destfile)
	// }
	return nil
}

type streamWriter struct {
	writer io.Writer
	buffer []byte
}

func (sw *streamWriter) Write(p []byte) (n int, err error) {
	var written int
	for len(p) > 0 {
		copySize := copy(sw.buffer, p)
		if copySize == 0 {
			break
		}

		wn, werr := sw.writer.Write(sw.buffer[:copySize])
		written += wn
		if werr != nil {
			return written, werr
		}

		p = p[copySize:]
	}
	return written, nil
}

func (p *Proxy) CreateModelFile(resp *http.Response, r *http.Request) (*os.File, error) {
	vars := mux.Vars(r)
	modelID := vars["model_id"]
	filename := vars["filename"]
	commit, etag, err := getCommitAndEtag(resp)
	if err != nil {
		return nil, err
	}
	blobDir := filepath.Join(p.path(modelID), "blobs")
	if _, err := os.Stat(blobDir); os.IsNotExist(err) {
		os.MkdirAll(blobDir, 0755)
	}
	blobPath := filepath.Join(blobDir, etag)
	file, err := os.Create(blobPath)
	destDir := filepath.Join(p.path(modelID), "snapshots", commit)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		os.MkdirAll(destDir, 0755)
	}
	destfile := filepath.Join(destDir, filename)
	symlinkOrRename(blobPath, destfile)
	return file, err
}

func (p *Proxy) CreateModelIndexFile(r *http.Request) (*os.File, error) {
	vars := mux.Vars(r)
	modelID := vars["model_id"]
	version := vars["version"]
	modelIndexPath := filepath.Join(p.path(modelID), ".modeindex")
	if _, err := os.Stat(modelIndexPath); err == nil {
		os.Remove(modelIndexPath)
	}
	file, err := os.Create(modelIndexPath)
	log.Printf("Downloading model index for %s", modelIndexPath)
	versionFileDir := filepath.Join(p.path(modelID), "refs")
	if _, err := os.Stat(versionFileDir); os.IsNotExist(err) {
		os.MkdirAll(versionFileDir, 0755)
	}
	versionFilePath := filepath.Join(versionFileDir, version)
	os.Create(versionFilePath)
	return file, err
}

func (p *Proxy) path(modelID string) string {
	dir := filepath.Join(p.baseDir, "hub", utils.ConvertModelIDToHFPath(modelID))
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
	return dir
}

func getCommitAndEtag(res *http.Response) (string, string, error) {
	commitHash := res.Header.Get("x-repo-commit")
	etag := res.Header.Get("x-linked-etag")
	if len(etag) == 0 {
		etag = res.Header.Get("etag")
	}
	etag = strings.ReplaceAll(etag, "\"", "")
	return commitHash, etag, nil
}
