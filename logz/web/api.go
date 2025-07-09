package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
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
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Code      int         `json:"code,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
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
	Level     string                 `json:"level" validate:"required,oneof=debug info warn error fatal panic"`
	Message   string                 `json:"message" validate:"required,min=1,max=10000"`
	TraceID   string                 `json:"trace_id,omitempty" validate:"omitempty,min=1,max=64"`
	SpanID    string                 `json:"span_id,omitempty" validate:"omitempty,min=1,max=32"`
	Service   string                 `json:"service,omitempty" validate:"omitempty,min=1,max=100"`
	Caller    string                 `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Timestamp time.Time              `json:"timestamp,omitempty"`
}

// FileInfoResponse 文件信息响应
type FileInfoResponse struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	SizeHuman    string    `json:"size_human"`
	ModTime      time.Time `json:"mod_time"`
	IsCompressed bool      `json:"is_compressed"`
	Path         string    `json:"path,omitempty"`
	Checksum     string    `json:"checksum,omitempty"`
	LineCount    int       `json:"line_count,omitempty"`
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

// 输入验证函数
func (api *APIServer) validateRequest(r *http.Request) error {
	// 检查Content-Type
	if r.Method == "POST" || r.Method == "PUT" {
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			return fmt.Errorf("unsupported content type: %s", contentType)
		}
	}
	
	// 检查请求大小
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB limit
	
	return nil
}

// 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
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

	// 验证请求
	if err := api.validateRequest(r); err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req LogQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendErrorResponse(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// 验证和设置默认值
	if req.Limit <= 0 || req.Limit > 10000 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// 验证时间范围
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() && req.StartTime.After(req.EndTime) {
		api.sendErrorResponse(w, "Start time cannot be after end time", http.StatusBadRequest)
		return
	}

	start := time.Now()
	query := logz.LogQuery{
		TraceID:   strings.TrimSpace(req.TraceID),
		SpanID:    strings.TrimSpace(req.SpanID),
		Level:     strings.ToLower(strings.TrimSpace(req.Level)),
		Service:   strings.TrimSpace(req.Service),
		Message:   strings.TrimSpace(req.Message),
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Limit:     req.Limit,
		Offset:    req.Offset,
		UseIndex:  req.UseIndex,
	}

	result, err := logz.QueryLogs(query, api.ws.logDir)
	if err != nil {
		api.sendErrorResponse(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 添加性能指标
	duration := time.Since(start)
	enhancedResult := map[string]interface{}{
		"result":     result,
		"duration":   duration.String(),
		"query_info": map[string]interface{}{
			"use_index": req.UseIndex,
			"limit":     req.Limit,
			"offset":    req.Offset,
		},
	}

	api.sendSuccessResponse(w, enhancedResult)
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

	// 验证请求
	if err := api.validateRequest(r); err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req LogWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendErrorResponse(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// 验证字段
	if err := api.validateLogWriteRequest(&req); err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 设置默认时间戳
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	// 清理和规范化字段
	req.Level = strings.ToLower(strings.TrimSpace(req.Level))
	req.Message = strings.TrimSpace(req.Message)
	req.Service = strings.TrimSpace(req.Service)

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
		api.sendErrorResponse(w, fmt.Sprintf("Failed to write log: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":   "Log entry written successfully",
		"entry_id":  fmt.Sprintf("%s-%d", req.Service, req.Timestamp.UnixNano()),
		"timestamp": req.Timestamp.Format(time.RFC3339),
	}

	api.sendSuccessResponseWithMessage(w, response, "Log written successfully")
}

// validateLogWriteRequest 验证日志写入请求
func (api *APIServer) validateLogWriteRequest(req *LogWriteRequest) error {
	// 验证级别
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true,
		"error": true, "fatal": true, "panic": true,
	}
	if !validLevels[strings.ToLower(req.Level)] {
		return fmt.Errorf("invalid log level: %s", req.Level)
	}

	// 验证消息长度
	if len(req.Message) == 0 {
		return fmt.Errorf("message cannot be empty")
	}
	if len(req.Message) > 10000 {
		return fmt.Errorf("message too long (max 10000 characters)")
	}

	// 验证其他字段长度
	if len(req.TraceID) > 64 {
		return fmt.Errorf("trace_id too long (max 64 characters)")
	}
	if len(req.SpanID) > 32 {
		return fmt.Errorf("span_id too long (max 32 characters)")
	}
	if len(req.Service) > 100 {
		return fmt.Errorf("service name too long (max 100 characters)")
	}

	return nil
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
	if err := api.validateFilename(filename); err != nil {
		api.sendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	filepath := filepath.Join(api.ws.logDir, filename)
	stat, err := os.Stat(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			api.sendErrorResponse(w, "File not found", http.StatusNotFound)
		} else {
			api.sendErrorResponse(w, fmt.Sprintf("Failed to get file info: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// 计算行数（对于文本文件）
	lineCount, _ := api.countFileLines(filepath)

	// 格式化文件大小
	sizeHuman := api.formatFileSize(stat.Size())

	fileInfo := FileInfoResponse{
		Name:         stat.Name(),
		Size:         stat.Size(),
		SizeHuman:    sizeHuman,
		ModTime:      stat.ModTime(),
		IsCompressed: strings.HasSuffix(filepath, ".gz"),
		Path:         filepath,
		LineCount:    lineCount,
	}

	api.sendSuccessResponse(w, fileInfo)
}

// validateFilename 验证文件名
func (api *APIServer) validateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if strings.Contains(filename, "..") {
		return fmt.Errorf("filename cannot contain '..'")
	}
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("filename cannot contain path separators")
	}
	if len(filename) > 255 {
		return fmt.Errorf("filename too long")
	}
	return nil
}

// countFileLines 计算文件行数
func (api *APIServer) countFileLines(filepath string) (int, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var count int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}

	return count, scanner.Err()
}

// formatFileSize 格式化文件大小
func (api *APIServer) formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	var i int
	floatSize := float64(size)
	for floatSize >= 1024 && i < len(units)-1 {
		floatSize /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", floatSize, units[i])
}

// sendSuccessResponse 发送成功响应
func (api *APIServer) sendSuccessResponse(w http.ResponseWriter, data interface{}) {
	api.sendResponse(w, true, data, "", http.StatusOK)
}

// sendSuccessResponseWithMessage 发送带消息的成功响应
func (api *APIServer) sendSuccessResponseWithMessage(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Code:      http.StatusOK,
		Timestamp: time.Now(),
		RequestID: generateRequestID(),
	}

	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse 发送错误响应
func (api *APIServer) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	api.sendResponse(w, false, nil, message, statusCode)
}

// sendResponse 统一响应处理
func (api *APIServer) sendResponse(w http.ResponseWriter, success bool, data interface{}, errorMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)

	response := APIResponse{
		Success:   success,
		Data:      data,
		Error:     errorMsg,
		Code:      statusCode,
		Timestamp: time.Now(),
		RequestID: generateRequestID(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
