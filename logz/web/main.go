package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HsiaoL1/trace/logz"
)

type WebServer struct {
	logDir      string
	port        string
	fileCache   map[string]*fileCacheEntry
	cacheMutex  sync.RWMutex
	server      *http.Server
	shutdownCh  chan struct{}
	clients     map[string]chan []byte // WebSocket clients for real-time logs
	clientsMutex sync.RWMutex
}

type fileCacheEntry struct {
	content   []string
	total     int
	lastMod   time.Time
	expiry    time.Time
}

type FileInfo struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	IsCompressed bool      `json:"is_compressed"`
}

type LogViewResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewWebServer(logDir, port string) *WebServer {
	return &WebServer{
		logDir:     logDir,
		port:       port,
		fileCache:  make(map[string]*fileCacheEntry),
		shutdownCh: make(chan struct{}),
		clients:    make(map[string]chan []byte),
	}
}

func (ws *WebServer) Start() error {
	// 启动缓存清理协程
	go ws.cacheCleanup()

	// 启动实时日志推送协程
	go ws.startLogStreaming()
	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %v", err)
	}

	// 确定模板和静态文件的路径
	// 如果当前在web目录下，直接使用templates和static
	// 如果在上级目录，使用web/templates和web/static
	templateDir := filepath.Join(currentDir, "templates")
	staticDir := filepath.Join(currentDir, "static")

	// 检查模板目录是否存在，如果不存在，尝试上级目录
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		templateDir = filepath.Join(currentDir, "web", "templates")
		staticDir = filepath.Join(currentDir, "web", "static")
	}

	// 再次检查模板目录是否存在
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return fmt.Errorf("模板目录不存在: %s", templateDir)
	}

	// 静态文件服务（支持gzip压缩）
	http.Handle("/static/", ws.gzipHandler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir)))))

	// 添加中间件
	http.HandleFunc("/api/files", ws.corsHandler(ws.rateLimitHandler(ws.logHandler(ws.getLogFiles))))
	http.HandleFunc("/api/search", ws.corsHandler(ws.rateLimitHandler(ws.logHandler(ws.searchLogs))))
	http.HandleFunc("/api/errors", ws.corsHandler(ws.rateLimitHandler(ws.logHandler(ws.getErrorLogs))))
	http.HandleFunc("/api/stats", ws.corsHandler(ws.rateLimitHandler(ws.logHandler(ws.getLogStats))))

	// 文件操作路由
	http.HandleFunc("/api/files/delete/", ws.corsHandler(ws.rateLimitHandler(ws.logHandler(ws.handleDeleteFile))))
	http.HandleFunc("/api/files/content/", ws.corsHandler(ws.rateLimitHandler(ws.logHandler(ws.handleGetContent))))
	http.HandleFunc("/api/files/upload", ws.corsHandler(ws.rateLimitHandler(ws.logHandler(ws.handleUploadFile))))
	http.HandleFunc("/api/logs/stream", ws.corsHandler(ws.handleLogStream))

	// 页面路由
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ws.indexPage(w, r, templateDir)
	})
	http.HandleFunc("/view/", func(w http.ResponseWriter, r *http.Request) {
		filename := strings.TrimPrefix(r.URL.Path, "/view/")
		ws.viewLogPage(w, r, filename, templateDir)
	})
	http.HandleFunc("/errors", func(w http.ResponseWriter, r *http.Request) {
		ws.errorsPage(w, r, templateDir)
	})

	ws.server = &http.Server{
		Addr:           ":" + ws.port,
		Handler:        nil,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	fmt.Printf("日志管理Web服务器启动在 http://localhost:%s\n", ws.port)
	fmt.Printf("模板目录: %s\n", templateDir)
	fmt.Printf("静态文件目录: %s\n", staticDir)
	return ws.server.ListenAndServe()
}

func (ws *WebServer) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := strings.TrimPrefix(r.URL.Path, "/api/files/delete/")
	ws.deleteLogFile(w, r, filename)
}

func (ws *WebServer) handleGetContent(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/files/content/")
	ws.getLogContent(w, r, filename)
}

func (ws *WebServer) indexPage(w http.ResponseWriter, r *http.Request, templateDir string) {
	tmpl, err := template.ParseFiles(filepath.Join(templateDir, "index.html"))
	if err != nil {
		http.Error(w, fmt.Sprintf("解析模板失败: %v", err), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (ws *WebServer) viewLogPage(w http.ResponseWriter, r *http.Request, filename string, templateDir string) {
	tmpl, err := template.ParseFiles(filepath.Join(templateDir, "view.html"))
	if err != nil {
		http.Error(w, fmt.Sprintf("解析模板失败: %v", err), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Filename": filename,
	}
	tmpl.Execute(w, data)
}

func (ws *WebServer) errorsPage(w http.ResponseWriter, r *http.Request, templateDir string) {
	tmpl, err := template.ParseFiles(filepath.Join(templateDir, "errors.html"))
	if err != nil {
		http.Error(w, fmt.Sprintf("解析模板失败: %v", err), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (ws *WebServer) getLogFiles(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob(filepath.Join(ws.logDir, "*.log*"))
	if err != nil {
		ws.sendJSONResponse(w, false, nil, err.Error())
		return
	}

	var fileInfos []FileInfo
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			continue
		}

		fileInfo := FileInfo{
			Name:         filepath.Base(file),
			Size:         stat.Size(),
			ModTime:      stat.ModTime(),
			IsCompressed: strings.HasSuffix(file, ".gz"),
		}
		fileInfos = append(fileInfos, fileInfo)
	}

	ws.sendJSONResponse(w, true, fileInfos, "")
}

func (ws *WebServer) deleteLogFile(w http.ResponseWriter, r *http.Request, filename string) {
	// 安全检查：确保文件名不包含路径遍历
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		ws.sendJSONResponse(w, false, nil, "无效的文件名")
		return
	}

	filepath := filepath.Join(ws.logDir, filename)
	if err := os.Remove(filepath); err != nil {
		ws.sendJSONResponse(w, false, nil, err.Error())
		return
	}

	ws.sendJSONResponse(w, true, "文件删除成功", "")
}

func (ws *WebServer) getLogContent(w http.ResponseWriter, r *http.Request, filename string) {
	// 安全检查
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		ws.sendJSONResponse(w, false, nil, "无效的文件名")
		return
	}

	filepath := filepath.Join(ws.logDir, filename)

	// 获取查询参数
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	search := r.URL.Query().Get("search")

	limit := 1000 // 默认限制
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	content, total, err := ws.readLogFile(filepath, limit, offset, search)
	if err != nil {
		ws.sendJSONResponse(w, false, nil, err.Error())
		return
	}

	result := map[string]interface{}{
		"content": content,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	}

	ws.sendJSONResponse(w, true, result, "")
}

func (ws *WebServer) searchLogs(w http.ResponseWriter, r *http.Request) {
	var request struct {
		TraceID   string    `json:"trace_id"`
		SpanID    string    `json:"span_id"`
		Level     string    `json:"level"`
		Service   string    `json:"service"`
		Message   string    `json:"message"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		Limit     int       `json:"limit"`
		Offset    int       `json:"offset"`
		UseIndex  bool      `json:"use_index"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		ws.sendJSONResponse(w, false, nil, err.Error())
		return
	}

	query := logz.LogQuery{
		TraceID:   request.TraceID,
		SpanID:    request.SpanID,
		Level:     request.Level,
		Service:   request.Service,
		Message:   request.Message,
		StartTime: request.StartTime,
		EndTime:   request.EndTime,
		Limit:     request.Limit,
		Offset:    request.Offset,
		UseIndex:  request.UseIndex,
	}

	result, err := logz.QueryLogs(query, ws.logDir)
	if err != nil {
		ws.sendJSONResponse(w, false, nil, err.Error())
		return
	}

	ws.sendJSONResponse(w, true, result, "")
}

func (ws *WebServer) getErrorLogs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	query := logz.LogQuery{
		Level:    "error",
		Limit:    limit,
		Offset:   offset,
		UseIndex: true,
	}

	result, err := logz.QueryLogs(query, ws.logDir)
	if err != nil {
		ws.sendJSONResponse(w, false, nil, err.Error())
		return
	}

	ws.sendJSONResponse(w, true, result, "")
}

func (ws *WebServer) getLogStats(w http.ResponseWriter, r *http.Request) {
	stats, err := logz.GetLogStats(ws.logDir)
	if err != nil {
		ws.sendJSONResponse(w, false, nil, err.Error())
		return
	}

	ws.sendJSONResponse(w, true, stats, "")
}

func (ws *WebServer) readLogFile(filepath string, limit, offset int, search string) ([]string, int, error) {
	// 检查缓存
	cacheKey := fmt.Sprintf("%s:%d:%d:%s", filepath, limit, offset, search)
	ws.cacheMutex.RLock()
	if entry, exists := ws.fileCache[cacheKey]; exists && time.Now().Before(entry.expiry) {
		stat, err := os.Stat(filepath)
		if err == nil && !stat.ModTime().After(entry.lastMod) {
			ws.cacheMutex.RUnlock()
			return entry.content, entry.total, nil
		}
	}
	ws.cacheMutex.RUnlock()

	// 读取文件
	content, total, err := ws.readFileContent(filepath, limit, offset, search)
	if err != nil {
		return nil, 0, err
	}

	// 更新缓存
	ws.cacheMutex.Lock()
	stat, _ := os.Stat(filepath)
	ws.fileCache[cacheKey] = &fileCacheEntry{
		content: content,
		total:   total,
		lastMod: stat.ModTime(),
		expiry:  time.Now().Add(5 * time.Minute), // 5分钟缓存
	}
	ws.cacheMutex.Unlock()

	return content, total, nil
}

func (ws *WebServer) readFileContent(filepath string, limit, offset int, search string) ([]string, int, error) {
	// 支持压缩文件
	var reader *bufio.Scanner
	file, err := os.Open(filepath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	if strings.HasSuffix(filepath, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, 0, err
		}
		defer gzReader.Close()
		reader = bufio.NewScanner(gzReader)
	} else {
		reader = bufio.NewScanner(file)
	}

	// 设置更大的缓冲区
	buf := make([]byte, 0, 64*1024)
	reader.Buffer(buf, 1024*1024)

	var lines []string
	var total int
	var matched int

	for reader.Scan() {
		line := reader.Text()
		total++

		// 应用搜索过滤
		if search != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(search)) {
			continue
		}

		// 应用分页
		if matched >= offset && len(lines) < limit {
			lines = append(lines, line)
		}
		matched++
	}

	return lines, total, reader.Err()
}

func (ws *WebServer) sendJSONResponse(w http.ResponseWriter, success bool, data interface{}, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")

	response := LogViewResponse{
		Success: success,
		Data:    data,
		Error:   errorMsg,
	}

	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) getLogFilesList() ([]FileInfo, error) {
	files, err := filepath.Glob(filepath.Join(ws.logDir, "*.log*"))
	if err != nil {
		return nil, err
	}
	var fileInfos []FileInfo
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfo := FileInfo{
			Name:         filepath.Base(file),
			Size:         stat.Size(),
			ModTime:      stat.ModTime(),
			IsCompressed: strings.HasSuffix(file, ".gz"),
		}
		fileInfos = append(fileInfos, fileInfo)
	}
	return fileInfos, nil
}

// 中间件函数
func (ws *WebServer) corsHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next(w, r)
	}
}

func (ws *WebServer) rateLimitHandler(next http.HandlerFunc) http.HandlerFunc {
	var requests = make(map[string][]time.Time)
	var mutex sync.Mutex
	
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		now := time.Now()
		
		mutex.Lock()
		// 清理过期的请求记录
		if times, exists := requests[clientIP]; exists {
			var validTimes []time.Time
			for _, t := range times {
				if now.Sub(t) < time.Minute {
					validTimes = append(validTimes, t)
				}
			}
			requests[clientIP] = validTimes
		}
		
		// 检查速率限制（每分钟100次请求）
		if len(requests[clientIP]) >= 100 {
			mutex.Unlock()
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		// 记录当前请求
		requests[clientIP] = append(requests[clientIP], now)
		mutex.Unlock()
		
		next(w, r)
	}
}

func (ws *WebServer) logHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// 创建响应记录器
		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		
		next(rec, r)
		
		// 记录请求日志
		duration := time.Since(start)
		log.Printf("%s %s %d %v %s", r.Method, r.URL.Path, rec.statusCode, duration, r.RemoteAddr)
	}
}

func (ws *WebServer) gzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		
		w.Header().Set("Content-Encoding", "gzip")
		gzWriter := gzip.NewWriter(w)
		defer gzWriter.Close()
		
		gzResponseWriter := &gzipResponseWriter{writer: gzWriter, ResponseWriter: w}
		next.ServeHTTP(gzResponseWriter, r)
	})
}

// 响应记录器
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Gzip响应写入器
type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *gzipResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// 缓存清理
func (ws *WebServer) cacheCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ws.cacheMutex.Lock()
			now := time.Now()
			for key, entry := range ws.fileCache {
				if now.After(entry.expiry) {
					delete(ws.fileCache, key)
				}
			}
			ws.cacheMutex.Unlock()
		case <-ws.shutdownCh:
			return
		}
	}
}

// 实时日志流
func (ws *WebServer) startLogStreaming() {
	// 这里可以实现实时日志推送逻辑
	// 例如监控日志文件变化，推送到WebSocket客户端
}

// 处理日志流连接
func (ws *WebServer) handleLogStream(w http.ResponseWriter, r *http.Request) {
	// 实现WebSocket连接处理逻辑
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	
	// 发送初始消息
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	
	// 这里可以实现具体的流式推送逻辑
}

// 处理文件上传
func (ws *WebServer) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 限制上传文件大小为10MB
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		ws.sendJSONResponse(w, false, nil, "解析上传文件失败")
		return
	}
	
	file, handler, err := r.FormFile("file")
	if err != nil {
		ws.sendJSONResponse(w, false, nil, "获取上传文件失败")
		return
	}
	defer file.Close()
	
	// 验证文件类型
	if !strings.HasSuffix(handler.Filename, ".log") && !strings.HasSuffix(handler.Filename, ".log.gz") {
		ws.sendJSONResponse(w, false, nil, "只支持.log和.log.gz文件")
		return
	}
	
	// 保存文件
	dstPath := filepath.Join(ws.logDir, handler.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		ws.sendJSONResponse(w, false, nil, "创建文件失败")
		return
	}
	defer dst.Close()
	
	if _, err := io.Copy(dst, file); err != nil {
		ws.sendJSONResponse(w, false, nil, "保存文件失败")
		return
	}
	
	ws.sendJSONResponse(w, true, map[string]string{"message": "文件上传成功"}, "")
}

// 优雅关闭
func (ws *WebServer) Shutdown(ctx context.Context) error {
	close(ws.shutdownCh)
	return ws.server.Shutdown(ctx)
}

func main() {
	logDir := "logs"
	port := "8080"

	// 从环境变量读取配置
	if envLogDir := os.Getenv("LOG_DIR"); envLogDir != "" {
		logDir = envLogDir
	}
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		return
	}

	server := NewWebServer(logDir, port)
	if err := server.Start(); err != nil {
		fmt.Printf("启动Web服务器失败: %v\n", err)
	}
}
