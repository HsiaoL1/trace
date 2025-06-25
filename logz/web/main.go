package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/HsiaoL1/trace/logz"
)

type WebServer struct {
	logDir string
	port   string
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
		logDir: logDir,
		port:   port,
	}
}

func (ws *WebServer) Start() error {
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

	// 静态文件服务
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// 原有的Web路由
	http.HandleFunc("/api/files", ws.getLogFiles)
	http.HandleFunc("/api/search", ws.searchLogs)
	http.HandleFunc("/api/errors", ws.getErrorLogs)
	http.HandleFunc("/api/stats", ws.getLogStats)

	// 文件操作路由
	http.HandleFunc("/api/files/delete/", ws.handleDeleteFile)
	http.HandleFunc("/api/files/content/", ws.handleGetContent)

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

	fmt.Printf("日志管理Web服务器启动在 http://localhost:%s\n", ws.port)
	fmt.Printf("模板目录: %s\n", templateDir)
	fmt.Printf("静态文件目录: %s\n", staticDir)
	return http.ListenAndServe(":"+ws.port, nil)
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
	file, err := os.Open(filepath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	var lines []string
	var total int
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		total++

		// 应用搜索过滤
		if search != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(search)) {
			continue
		}

		// 应用分页
		if len(lines) < limit && len(lines) >= offset {
			lines = append(lines, line)
		}
	}

	return lines, total, scanner.Err()
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

func main() {
	logDir := "logs"
	port := "8080"

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
