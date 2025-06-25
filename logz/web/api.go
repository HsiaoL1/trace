package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/HsiaoL1/trace/logz"
)

// APIResponse 标准API响应格式
type APIResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// LogQueryRequest 日志查询请求
type LogQueryRequest struct {
	TraceID   string    `json:"trace_id,omitempty"`
	SpanID    string    `json:"span_id,omitempty"`
	Level     string    `json:"level,omitempty"`
	Service   string    `json:"service,omitempty"`
	Message   string    `json:"message,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
	UseIndex  bool      `json:"use_index,omitempty"`
}

// LogWriteRequest 日志写入请求
type LogWriteRequest struct {
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	TraceID   string         `json:"trace_id,omitempty"`
	SpanID    string         `json:"span_id,omitempty"`
	Service   string         `json:"service,omitempty"`
	Caller    string         `json:"caller,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
	Timestamp time.Time      `json:"timestamp,omitempty"`
}

// FileInfoResponse 文件信息响应
type FileInfoResponse struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	IsCompressed bool      `json:"is_compressed"`
	Path         string    `json:"path,omitempty"`
}

// StatsResponse 统计信息响应
type StatsResponse struct {
	TotalFiles int       `json:"total_files"`
	TotalSize  int64     `json:"total_size"`
	OldestFile string    `json:"oldest_file"`
	NewestFile string    `json:"newest_file"`
	OldestTime time.Time `json:"oldest_time"`
	NewestTime time.Time `json:"newest_time"`
}

// APIServer API服务器
type APIServer struct {
	ws *WebServer
}

// NewAPIServer 创建API服务器
func NewAPIServer(ws *WebServer) *APIServer {
	return &APIServer{ws: ws}
}

// SetupAPIRoutes 设置API路由
func (api *APIServer) SetupAPIRoutes() {
	// 日志查询API
	http.HandleFunc("/api/v1/logs/search", api.handleLogSearch)
	http.HandleFunc("/api/v1/logs/trace/", api.handleLogSearchByTraceID)
	http.HandleFunc("/api/v1/logs/span/", api.handleLogSearchBySpanID)
	http.HandleFunc("/api/v1/logs/level/", api.handleLogSearchByLevel)
	http.HandleFunc("/api/v1/logs/service/", api.handleLogSearchByService)
	http.HandleFunc("/api/v1/logs/errors", api.handleErrorLogs)

	// 日志写入API
	http.HandleFunc("/api/v1/logs/write", api.handleLogWrite)

	// 文件管理API
	http.HandleFunc("/api/v1/files", api.handleGetFiles)
	http.HandleFunc("/api/v1/files/", api.handleFileOperations)
	http.HandleFunc("/api/v1/files/content/", api.handleGetFileContent)

	// 统计信息API
	http.HandleFunc("/api/v1/stats", api.handleGetStats)

	// 健康检查API
	http.HandleFunc("/api/v1/health", api.handleHealthCheck)
}

// handleLogSearch 处理日志搜索
func (api *APIServer) handleLogSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LogQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 设置默认值
	if req.Limit == 0 {
		req.Limit = 100
	}

	query := logz.LogQuery{
		TraceID:   req.TraceID,
		SpanID:    req.SpanID,
		Level:     req.Level,
		Service:   req.Service,
		Message:   req.Message,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Limit:     req.Limit,
		Offset:    req.Offset,
		UseIndex:  req.UseIndex,
	}

	result, err := logz.QueryLogs(query, api.ws.logDir)
	if err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.sendSuccessResponse(w, result)
}

// handleLogSearchByTraceID 根据TraceID搜索日志
func (api *APIServer) handleLogSearchByTraceID(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	traceID := strings.TrimPrefix(r.URL.Path, "/api/v1/logs/trace/")
	if traceID == "" {
		api.sendErrorResponse(w, "TraceID is required", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	result, err := logz.QueryLogsByTraceID(traceID, api.ws.logDir, limit, offset)
	if err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.sendSuccessResponse(w, result)
}

// handleLogSearchBySpanID 根据SpanID搜索日志
func (api *APIServer) handleLogSearchBySpanID(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	spanID := strings.TrimPrefix(r.URL.Path, "/api/v1/logs/span/")
	if spanID == "" {
		api.sendErrorResponse(w, "SpanID is required", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	result, err := logz.QueryLogsBySpanID(spanID, api.ws.logDir, limit, offset)
	if err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.sendSuccessResponse(w, result)
}

// handleLogSearchByLevel 根据日志级别搜索
func (api *APIServer) handleLogSearchByLevel(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	level := strings.TrimPrefix(r.URL.Path, "/api/v1/logs/level/")
	if level == "" {
		api.sendErrorResponse(w, "Level is required", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	result, err := logz.QueryLogsByLevel(level, api.ws.logDir, limit, offset)
	if err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.sendSuccessResponse(w, result)
}

// handleLogSearchByService 根据服务名搜索
func (api *APIServer) handleLogSearchByService(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	service := strings.TrimPrefix(r.URL.Path, "/api/v1/logs/service/")
	if service == "" {
		api.sendErrorResponse(w, "Service name is required", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	result, err := logz.QueryLogsByService(service, api.ws.logDir, limit, offset)
	if err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.sendSuccessResponse(w, result)
}

// handleErrorLogs 获取错误日志
func (api *APIServer) handleErrorLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 100
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	result, err := logz.QueryLogsByLevel("error", api.ws.logDir, limit, offset)
	if err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.sendSuccessResponse(w, result)
}

// handleLogWrite 处理日志写入
func (api *APIServer) handleLogWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LogWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证必需字段
	if req.Level == "" || req.Message == "" {
		api.sendErrorResponse(w, "Level and message are required", http.StatusBadRequest)
		return
	}

	// 设置默认时间戳
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	// 创建日志条目
	entry := logz.LogEntry{
		Timestamp: req.Timestamp.Format(time.RFC3339),
		Level:     req.Level,
		Message:   req.Message,
		TraceID:   req.TraceID,
		SpanID:    req.SpanID,
		Service:   req.Service,
		Caller:    req.Caller,
		Fields:    req.Fields,
	}

	// 写入到聚合器
	if err := logz.WriteToAggregator(entry); err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.sendSuccessResponse(w, map[string]string{"message": "Log entry written successfully"})
}

// handleGetFiles 获取文件列表
func (api *APIServer) handleGetFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, err := api.ws.getLogFilesList()
	if err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.sendSuccessResponse(w, files)
}

// handleFileOperations 处理文件操作
func (api *APIServer) handleFileOperations(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/v1/files/")

	switch r.Method {
	case "DELETE":
		api.handleDeleteFile(w, r, filename)
	case "GET":
		api.handleGetFileInfo(w, r, filename)
	default:
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetFileContent 获取文件内容
func (api *APIServer) handleGetFileContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := strings.TrimPrefix(r.URL.Path, "/api/v1/files/content/")
	if filename == "" {
		api.sendErrorResponse(w, "Filename is required", http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 1000
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	search := r.URL.Query().Get("search")

	content, total, err := api.ws.readLogFile(filepath.Join(api.ws.logDir, filename), limit, offset, search)
	if err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"content":  content,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"filename": filename,
	}

	api.sendSuccessResponse(w, result)
}

// handleGetStats 获取统计信息
func (api *APIServer) handleGetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := logz.GetLogStats(api.ws.logDir)
	if err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.sendSuccessResponse(w, stats)
}

// handleHealthCheck 健康检查
func (api *APIServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		api.sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "log-management-api",
		"version":   "1.0.0",
	}

	api.sendSuccessResponse(w, health)
}

// handleDeleteFile 处理文件删除
func (api *APIServer) handleDeleteFile(w http.ResponseWriter, r *http.Request, filename string) {
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		api.sendErrorResponse(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filepath := filepath.Join(api.ws.logDir, filename)
	if err := os.Remove(filepath); err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api.sendSuccessResponse(w, map[string]string{"message": "File deleted successfully"})
}

// handleGetFileInfo 获取文件信息
func (api *APIServer) handleGetFileInfo(w http.ResponseWriter, r *http.Request, filename string) {
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		api.sendErrorResponse(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filepath := filepath.Join(api.ws.logDir, filename)
	stat, err := os.Stat(filepath)
	if err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusNotFound)
		return
	}

	fileInfo := FileInfoResponse{
		Name:         stat.Name(),
		Size:         stat.Size(),
		ModTime:      stat.ModTime(),
		IsCompressed: strings.HasSuffix(filepath, ".gz"),
		Path:         filepath,
	}

	api.sendSuccessResponse(w, fileInfo)
}

// sendSuccessResponse 发送成功响应
func (api *APIServer) sendSuccessResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := APIResponse{
		Success: true,
		Data:    data,
		Code:    http.StatusOK,
	}

	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse 发送错误响应
func (api *APIServer) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := APIResponse{
		Success: false,
		Error:   message,
		Code:    statusCode,
	}

	json.NewEncoder(w).Encode(response)
}
