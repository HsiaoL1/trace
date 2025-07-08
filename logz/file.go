package logz

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
)

// 这里需要对日志文件进行聚合，以及对聚合之后的文件进行查询
// 比如通过traceID进行查询，或者通过时间范围进行查询
// 需要删除一个星期之前的日志文件

// LogEntry 日志条目结构
type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"msg"`
	TraceID   string         `json:"trace_id,omitempty"`
	SpanID    string         `json:"span_id,omitempty"`
	Caller    string         `json:"caller,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
	Service   string         `json:"service,omitempty"`
	File      string         `json:"file,omitempty"`
	FileID    string         `json:"file_id,omitempty"` // 文件标识
	Offset    int64          `json:"offset,omitempty"`  // 在文件中的偏移量
}

// LogAggregator 日志聚合器
type LogAggregator struct {
	outputDir     string
	serviceName   string
	rotationSize  int64
	maxBackups    int
	aggregateFile *os.File
	writer        *bufio.Writer
	mutex         sync.RWMutex
	lastRotation  time.Time
	currentFileID string
	currentOffset int64

	// 索引相关
	indexDB    *bbolt.DB
	indexMutex sync.RWMutex

	// 批量写入
	batchSize     int
	batchBuffer   []LogEntry
	batchMutex    sync.Mutex
	batchTicker   *time.Ticker
	flushInterval time.Duration

	// 压缩相关
	compressAfter time.Duration
	compressMutex sync.Mutex

	// 生命周期管理
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	closed    bool
	closeMutex sync.Mutex

	// 索引工作队列
	indexQueue   chan LogEntry
	indexWorkers int
}

// LogQuery 日志查询条件
type LogQuery struct {
	TraceID   string    `json:"trace_id,omitempty"`
	SpanID    string    `json:"span_id,omitempty"`
	Level     string    `json:"level,omitempty"`
	Service   string    `json:"service,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Message   string    `json:"message,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
	UseIndex  bool      `json:"use_index,omitempty"` // 是否使用索引
}

// LogQueryResult 查询结果
type LogQueryResult struct {
	Entries []LogEntry `json:"entries"`
	Total   int        `json:"total"`
	Limit   int        `json:"limit"`
	Offset  int        `json:"offset"`
}

// IndexEntry 索引条目
type IndexEntry struct {
	FileID string `json:"file_id"`
	Offset int64  `json:"offset"`
	Size   int    `json:"size"`
}

// NewLogAggregator 创建新的日志聚合器
func NewLogAggregator(outputDir, serviceName string, rotationSize int64, maxBackups int) (*LogAggregator, error) {
	// 参数验证
	if outputDir == "" {
		return nil, errors.New("输出目录不能为空")
	}
	if serviceName == "" {
		return nil, errors.New("服务名不能为空")
	}
	if rotationSize <= 0 {
		rotationSize = 100 * 1024 * 1024 // 100MB
	}
	if maxBackups <= 0 {
		maxBackups = 10
	}

	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志聚合目录失败: %w", err)
	}

	// 创建索引目录
	indexDir := filepath.Join(outputDir, "index")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return nil, fmt.Errorf("创建索引目录失败: %w", err)
	}

	// 打开索引数据库
	indexDB, err := bbolt.Open(filepath.Join(indexDir, serviceName+".db"), 0600, &bbolt.Options{
		Timeout: 5 * time.Second,
		NoSync:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("打开索引数据库失败: %w", err)
	}

	// 初始化索引桶
	err = indexDB.Update(func(tx *bbolt.Tx) error {
		buckets := []string{"trace_id", "span_id", "level", "service", "time"}
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("创建索引桶%s失败: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		indexDB.Close()
		return nil, err
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	aggregator := &LogAggregator{
		outputDir:     outputDir,
		serviceName:   serviceName,
		rotationSize:  rotationSize,
		maxBackups:    maxBackups,
		lastRotation:  time.Now(),
		indexDB:       indexDB,
		batchSize:     100,
		batchBuffer:   make([]LogEntry, 0, 100),
		flushInterval: 5 * time.Second,
		compressAfter: 24 * time.Hour,
		ctx:           ctx,
		cancel:        cancel,
		done:          make(chan struct{}),
		indexQueue:    make(chan LogEntry, 1000), // 缓冲队列
		indexWorkers:  2,                        // 索引工作线程数
	}

	// 初始化聚合文件
	if err := aggregator.initializeFile(); err != nil {
		cancel()
		indexDB.Close()
		return nil, err
	}

	// 启动后台任务
	aggregator.startBackgroundTasks()

	return aggregator, nil
}

// initializeFile 初始化聚合文件
func (la *LogAggregator) initializeFile() error {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	// 关闭现有文件
	if la.writer != nil {
		if err := la.writer.Flush(); err != nil {
			return fmt.Errorf("刷新缓冲区失败: %w", err)
		}
	}
	if la.aggregateFile != nil {
		if err := la.aggregateFile.Close(); err != nil {
			return fmt.Errorf("关闭文件失败: %w", err)
		}
	}

	// 生成文件ID
	now := time.Now()
	la.currentFileID = fmt.Sprintf("%s_%s_%03d", la.serviceName, now.Format("2006-01-02"), la.getFileSequence(now))
	la.currentOffset = 0

	// 创建新的聚合文件
	filename := la.currentFileID + ".log"
	filePath := filepath.Join(la.outputDir, filename)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("创建聚合日志文件失败: %w", err)
	}

	// 获取文件当前大小作为偏移量
	if stat, err := file.Stat(); err == nil {
		la.currentOffset = stat.Size()
	}

	la.aggregateFile = file
	la.writer = bufio.NewWriterSize(file, 32*1024) // 32KB缓冲
	return nil
}

// getFileSequence 获取当天的文件序列号
func (la *LogAggregator) getFileSequence(date time.Time) int {
	pattern := filepath.Join(la.outputDir, fmt.Sprintf("%s_%s_*.log", la.serviceName, date.Format("2006-01-02")))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return 1
	}
	
	// 过滤压缩文件
	var validFiles []string
	for _, file := range files {
		if !strings.HasSuffix(file, ".gz") {
			validFiles = append(validFiles, file)
		}
	}
	
	return len(validFiles) + 1
}

// WriteLog 写入日志到聚合文件
func (la *LogAggregator) WriteLog(entry LogEntry) error {
	// 检查是否已关闭
	la.closeMutex.Lock()
	if la.closed {
		la.closeMutex.Unlock()
		return errors.New("聚合器已关闭")
	}
	la.closeMutex.Unlock()

	la.batchMutex.Lock()
	defer la.batchMutex.Unlock()

	// 设置文件信息
	entry.FileID = la.currentFileID
	entry.Offset = la.currentOffset
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339)
	}

	// 添加到批量缓冲区
	la.batchBuffer = append(la.batchBuffer, entry)

	// 检查是否需要轮转文件
	if la.shouldRotate() {
		if err := la.rotateFile(); err != nil {
			return fmt.Errorf("轮转文件失败: %w", err)
		}
	}

	// 检查是否需要批量写入
	if len(la.batchBuffer) >= la.batchSize {
		return la.flushBatch()
	}

	return nil
}

// flushBatch 刷新批量缓冲区
func (la *LogAggregator) flushBatch() error {
	if len(la.batchBuffer) == 0 {
		return nil
	}

	la.mutex.Lock()
	defer la.mutex.Unlock()

	// 为本次批量做备份
	batchToWrite := make([]LogEntry, len(la.batchBuffer))
	copy(batchToWrite, la.batchBuffer)

	// 先清空缓冲区，避免长时间锁定
	la.batchBuffer = la.batchBuffer[:0]

	// 写入所有条目
	for _, entry := range batchToWrite {
		// 序列化日志条目
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("序列化日志条目失败: %w", err)
		}

		// 写入文件
		line := append(data, '\n')
		if _, err := la.writer.Write(line); err != nil {
			return fmt.Errorf("写入日志文件失败: %w", err)
		}

		// 更新偏移量
		la.currentOffset += int64(len(line))

		// 异步添加到索引队列
		select {
		case la.indexQueue <- entry:
		case <-la.ctx.Done():
			return la.ctx.Err()
		default:
			// 队列已满，跳过索引
		}
	}

	// 刷新缓冲区
	if err := la.writer.Flush(); err != nil {
		return fmt.Errorf("刷新文件缓冲区失败: %w", err)
	}

	return nil
}

// addToIndex 添加到索引（在工作线程中调用）
func (la *LogAggregator) addToIndex(entry LogEntry) error {
	return la.indexDB.Update(func(tx *bbolt.Tx) error {
		value := fmt.Sprintf("%s:%d", entry.FileID, entry.Offset)
		
		// 添加TraceID索引
		if entry.TraceID != "" {
			if bucket := tx.Bucket([]byte("trace_id")); bucket != nil {
				if err := bucket.Put([]byte(entry.TraceID), []byte(value)); err != nil {
					return fmt.Errorf("添加TraceID索引失败: %w", err)
				}
			}
		}

		// 添加SpanID索引
		if entry.SpanID != "" {
			if bucket := tx.Bucket([]byte("span_id")); bucket != nil {
				if err := bucket.Put([]byte(entry.SpanID), []byte(value)); err != nil {
					return fmt.Errorf("添加SpanID索引失败: %w", err)
				}
			}
		}

		// 添加级别索引
		if entry.Level != "" {
			if bucket := tx.Bucket([]byte("level")); bucket != nil {
				key := strings.ToLower(entry.Level)
				if err := bucket.Put([]byte(key), []byte(value)); err != nil {
					return fmt.Errorf("添加级别索引失败: %w", err)
				}
			}
		}

		// 添加服务索引
		if entry.Service != "" {
			if bucket := tx.Bucket([]byte("service")); bucket != nil {
				if err := bucket.Put([]byte(entry.Service), []byte(value)); err != nil {
					return fmt.Errorf("添加服务索引失败: %w", err)
				}
			}
		}

		// 添加时间索引
		if entry.Timestamp != "" {
			if bucket := tx.Bucket([]byte("time")); bucket != nil {
				if err := bucket.Put([]byte(entry.Timestamp), []byte(value)); err != nil {
					return fmt.Errorf("添加时间索引失败: %w", err)
				}
			}
		}

		return nil
	})
}

// shouldRotate 检查是否需要轮转文件
func (la *LogAggregator) shouldRotate() bool {
	// 检查文件大小
	if la.aggregateFile != nil {
		if stat, err := la.aggregateFile.Stat(); err == nil {
			if stat.Size() >= la.rotationSize {
				return true
			}
		}
	}

	// 检查日期变化（跨天轮转）
	now := time.Now()
	return now.Day() != la.lastRotation.Day() || now.Month() != la.lastRotation.Month() || now.Year() != la.lastRotation.Year()
}

// rotateFile 轮转文件
func (la *LogAggregator) rotateFile() error {
	// 刷新批量缓冲区
	if err := la.flushBatch(); err != nil {
		return fmt.Errorf("轮转前刷新失败: %w", err)
	}

	// 刷新并关闭当前文件
	if la.writer != nil {
		if err := la.writer.Flush(); err != nil {
			return fmt.Errorf("刷新文件失败: %w", err)
		}
	}
	if la.aggregateFile != nil {
		if err := la.aggregateFile.Close(); err != nil {
			return fmt.Errorf("关闭文件失败: %w", err)
		}
	}

	// 清理旧文件
	if err := la.cleanupOldFiles(); err != nil {
		// 清理失败不影响轮转操作
		fmt.Fprintf(os.Stderr, "[清理旧文件错误] %v\n", err)
	}

	// 初始化新文件
	if err := la.initializeFile(); err != nil {
		return fmt.Errorf("初始化新文件失败: %w", err)
	}

	// 更新轮转时间
	la.lastRotation = time.Now()
	return nil
}

// cleanupOldFiles 清理旧文件
func (la *LogAggregator) cleanupOldFiles() error {
	// 删除一周前的文件
	cutoffTime := time.Now().AddDate(0, 0, -7)

	files, err := filepath.Glob(filepath.Join(la.outputDir, la.serviceName+"_*.log"))
	if err != nil {
		return err
	}

	for _, file := range files {
		if stat, err := os.Stat(file); err == nil {
			if stat.ModTime().Before(cutoffTime) {
				os.Remove(file)
			}
		}
	}

	return nil
}

// startBackgroundTasks 启动后台任务
func (la *LogAggregator) startBackgroundTasks() {
	// 启动索引工作线程
	for i := 0; i < la.indexWorkers; i++ {
		go la.indexWorker()
	}

	// 启动定时刷新任务
	la.batchTicker = time.NewTicker(la.flushInterval)
	go la.flushTask()

	// 启动清理和压缩任务
	go la.maintenanceTask()
}

// indexWorker 索引工作线程
func (la *LogAggregator) indexWorker() {
	for {
		select {
		case entry := <-la.indexQueue:
			if err := la.addToIndex(entry); err != nil {
				// 索引失败不影响主流程，只记录错误
				fmt.Fprintf(os.Stderr, "[索引错误] %v\n", err)
			}
		case <-la.ctx.Done():
			return
		}
	}
}

// flushTask 定时刷新任务
func (la *LogAggregator) flushTask() {
	defer la.batchTicker.Stop()
	
	for {
		select {
		case <-la.batchTicker.C:
			if err := la.flushBatch(); err != nil {
				fmt.Fprintf(os.Stderr, "[刷新错误] %v\n", err)
			}
		case <-la.ctx.Done():
			return
		}
	}
}

// maintenanceTask 维护任务（清理和压缩）
func (la *LogAggregator) maintenanceTask() {
	maintenanceTicker := time.NewTicker(1 * time.Hour)
	defer maintenanceTicker.Stop()

	for {
		select {
		case <-maintenanceTicker.C:
			// 压缩旧文件
			la.compressOldFiles()
			
			// 清理过期文件
			if err := la.cleanupOldFiles(); err != nil {
				fmt.Fprintf(os.Stderr, "[清理错误] %v\n", err)
			}
		case <-la.ctx.Done():
			return
		}
	}
}

// compressOldFiles 压缩旧文件
func (la *LogAggregator) compressOldFiles() {
	la.compressMutex.Lock()
	defer la.compressMutex.Unlock()

	cutoffTime := time.Now().Add(-la.compressAfter)

	pattern := filepath.Join(la.outputDir, la.serviceName+"_*.log")
	files, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[获取文件列表错误] %v\n", err)
		return
	}

	for _, file := range files {
		// 跳过当前正在写入的文件
		if strings.Contains(file, la.currentFileID) {
			continue
		}

		stat, err := os.Stat(file)
		if err != nil {
			continue
		}

		// 检查文件是否过期且未压缩
		if stat.ModTime().Before(cutoffTime) && !strings.HasSuffix(file, ".gz") {
			if err := la.compressFile(file); err != nil {
				fmt.Fprintf(os.Stderr, "[压缩文件错误] %s: %v\n", file, err)
			}
		}
	}
}

// compressFile 压缩文件
func (la *LogAggregator) compressFile(filePath string) error {
	// 打开原文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 创建压缩文件
	gzPath := filePath + ".gz"
	gzFile, err := os.Create(gzPath)
	if err != nil {
		return fmt.Errorf("创建压缩文件失败: %w", err)
	}
	defer gzFile.Close()

	// 创建gzip writer
	gzWriter := gzip.NewWriter(gzFile)
	defer gzWriter.Close()

	// 复制内容
	_, err = io.Copy(gzWriter, file)
	if err != nil {
		// 清理已创建的压缩文件
		os.Remove(gzPath)
		return fmt.Errorf("压缩文件失败: %w", err)
	}

	// 确保数据写入磁盘
	if err := gzWriter.Close(); err != nil {
		os.Remove(gzPath)
		return fmt.Errorf("关闭压缩文件失败: %w", err)
	}
	if err := gzFile.Sync(); err != nil {
		os.Remove(gzPath)
		return fmt.Errorf("同步压缩文件失败: %w", err)
	}

	// 删除原文件
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("删除原文件失败: %w", err)
	}

	return nil
}

// Close 关闭聚合器
func (la *LogAggregator) Close() error {
	la.closeMutex.Lock()
	defer la.closeMutex.Unlock()
	
	if la.closed {
		return nil // 已经关闭
	}
	la.closed = true

	// 取消上下文，停止所有后台任务
	la.cancel()

	// 等待后台任务结束
	select {
	case <-la.done:
	case <-time.After(10 * time.Second):
		// 超时保护
	}

	// 最后一次刷新批量缓冲区
	la.batchMutex.Lock()
	la.flushBatch()
	la.batchMutex.Unlock()

	// 关闭文件
	la.mutex.Lock()
	if la.writer != nil {
		la.writer.Flush()
		la.writer = nil
	}
	if la.aggregateFile != nil {
		la.aggregateFile.Close()
		la.aggregateFile = nil
	}
	la.mutex.Unlock()

	// 关闭索引数据库
	if la.indexDB != nil {
		la.indexDB.Close()
		la.indexDB = nil
	}

	// 关闭索引队列
	close(la.indexQueue)

	// 关闭完成通知
	close(la.done)

	return nil
}

// QueryLogs 查询日志
func QueryLogs(query LogQuery, logDir string) (*LogQueryResult, error) {
	result := &LogQueryResult{
		Entries: make([]LogEntry, 0),
		Limit:   query.Limit,
		Offset:  query.Offset,
	}

	// 获取全局聚合器
	aggregator := GetGlobalAggregator()

	// 如果使用索引且查询条件简单，尝试使用索引
	if query.UseIndex && aggregator != nil && canUseIndex(query) {
		entries, err := queryWithIndex(query, logDir, aggregator)
		if err == nil {
			result.Entries = entries
			result.Total = len(entries)
			return result, nil
		}
	}

	// 回退到文件扫描
	return queryWithFileScan(query, logDir)
}

// canUseIndex 检查是否可以使用索引
func canUseIndex(query LogQuery) bool {
	// 只有单一条件查询才使用索引
	conditions := 0
	if query.TraceID != "" {
		conditions++
	}
	if query.SpanID != "" {
		conditions++
	}
	if query.Level != "" {
		conditions++
	}
	if query.Service != "" {
		conditions++
	}

	return conditions == 1
}

// queryWithIndex 使用索引查询
func queryWithIndex(query LogQuery, logDir string, aggregator *LogAggregator) ([]LogEntry, error) {
	var entries []LogEntry
	var bucketName string
	var key []byte

	// 确定查询的索引桶和键
	if query.TraceID != "" {
		bucketName = "trace_id"
		key = []byte(query.TraceID)
	} else if query.SpanID != "" {
		bucketName = "span_id"
		key = []byte(query.SpanID)
	} else if query.Level != "" {
		bucketName = "level"
		key = []byte(strings.ToLower(query.Level))
	} else if query.Service != "" {
		bucketName = "service"
		key = []byte(query.Service)
	}

	// 从索引中查找
	err := aggregator.indexDB.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("索引桶不存在")
		}

		value := bucket.Get(key)
		if value == nil {
			return fmt.Errorf("未找到匹配的索引")
		}

		// 解析索引值
		parts := strings.Split(string(value), ":")
		if len(parts) != 2 {
			return fmt.Errorf("索引格式错误")
		}

		fileID := parts[0]
		offset, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return err
		}

		// 从文件中读取日志条目
		entry, err := readLogEntry(filepath.Join(logDir, fileID+".log"), offset)
		if err != nil {
			return err
		}

		entries = append(entries, entry)
		return nil
	})

	return entries, err
}

// readLogEntry 从文件中读取指定偏移量的日志条目
func readLogEntry(filepath string, offset int64) (LogEntry, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return LogEntry{}, err
	}
	defer file.Close()

	// 定位到指定偏移量
	_, err = file.Seek(offset, 0)
	if err != nil {
		return LogEntry{}, err
	}

	// 读取一行
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return LogEntry{}, err
		}
		return entry, nil
	}

	return LogEntry{}, fmt.Errorf("无法读取日志条目")
}

// queryWithFileScan 使用文件扫描查询
func queryWithFileScan(query LogQuery, logDir string) (*LogQueryResult, error) {
	result := &LogQueryResult{
		Entries: make([]LogEntry, 0),
		Limit:   query.Limit,
		Offset:  query.Offset,
	}

	// 获取所有日志文件
	files, err := filepath.Glob(filepath.Join(logDir, "*.log"))
	if err != nil {
		return nil, fmt.Errorf("获取日志文件失败: %v", err)
	}

	// 按时间排序文件（最新的在前）
	sort.Slice(files, func(i, j int) bool {
		statI, _ := os.Stat(files[i])
		statJ, _ := os.Stat(files[j])
		return statI.ModTime().After(statJ.ModTime())
	})

	// 遍历文件进行查询
	for _, file := range files {
		entries, err := queryFile(file, query)
		if err != nil {
			continue // 跳过有问题的文件
		}

		result.Entries = append(result.Entries, entries...)
	}

	// 应用分页
	total := len(result.Entries)
	if query.Offset >= total {
		result.Entries = []LogEntry{}
	} else {
		end := query.Offset + query.Limit
		if end > total {
			end = total
		}
		result.Entries = result.Entries[query.Offset:end]
	}

	result.Total = total
	return result, nil
}

// queryFile 查询单个文件
func queryFile(filepath string, query LogQuery) ([]LogEntry, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // 跳过无效的JSON行
		}

		// 应用查询条件
		if !matchesQuery(entry, query) {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

// matchesQuery 检查日志条目是否匹配查询条件
func matchesQuery(entry LogEntry, query LogQuery) bool {
	// 检查TraceID
	if query.TraceID != "" && entry.TraceID != query.TraceID {
		return false
	}

	// 检查SpanID
	if query.SpanID != "" && entry.SpanID != query.SpanID {
		return false
	}

	// 检查日志级别
	if query.Level != "" && !strings.EqualFold(entry.Level, query.Level) {
		return false
	}

	// 检查服务名
	if query.Service != "" && entry.Service != query.Service {
		return false
	}

	// 检查消息内容
	if query.Message != "" {
		matched, _ := regexp.MatchString(query.Message, entry.Message)
		if !matched {
			return false
		}
	}

	// 检查时间范围
	if !query.StartTime.IsZero() || !query.EndTime.IsZero() {
		entryTime, err := time.Parse(time.RFC3339, entry.Timestamp)
		if err != nil {
			return false
		}

		if !query.StartTime.IsZero() && entryTime.Before(query.StartTime) {
			return false
		}

		if !query.EndTime.IsZero() && entryTime.After(query.EndTime) {
			return false
		}
	}

	return true
}

// CleanupOldLogs 清理旧日志文件
func CleanupOldLogs(logDir string, daysToKeep int) error {
	cutoffTime := time.Now().AddDate(0, 0, -daysToKeep)

	files, err := filepath.Glob(filepath.Join(logDir, "*.log"))
	if err != nil {
		return err
	}

	var deletedCount int
	for _, file := range files {
		if stat, err := os.Stat(file); err == nil {
			if stat.ModTime().Before(cutoffTime) {
				if err := os.Remove(file); err == nil {
					deletedCount++
				}
			}
		}
	}

	return nil
}

// GetLogStats 获取日志统计信息
func GetLogStats(logDir string) (map[string]any, error) {
	files, err := filepath.Glob(filepath.Join(logDir, "*.log"))
	if err != nil {
		return nil, err
	}

	stats := map[string]any{
		"total_files": len(files),
		"total_size":  int64(0),
		"oldest_file": "",
		"newest_file": "",
	}

	var oldestTime, newestTime time.Time
	var totalSize int64

	for _, file := range files {
		if stat, err := os.Stat(file); err == nil {
			totalSize += stat.Size()

			if oldestTime.IsZero() || stat.ModTime().Before(oldestTime) {
				oldestTime = stat.ModTime()
				stats["oldest_file"] = filepath.Base(file)
			}

			if newestTime.IsZero() || stat.ModTime().After(newestTime) {
				newestTime = stat.ModTime()
				stats["newest_file"] = filepath.Base(file)
			}
		}
	}

	stats["total_size"] = totalSize
	stats["oldest_time"] = oldestTime
	stats["newest_time"] = newestTime

	return stats, nil
}

// 全局聚合器实例
var globalAggregator *LogAggregator
var aggregatorMutex sync.Mutex

// SetGlobalAggregator 设置全局聚合器
func SetGlobalAggregator(aggregator *LogAggregator) {
	aggregatorMutex.Lock()
	defer aggregatorMutex.Unlock()
	globalAggregator = aggregator
}

// GetGlobalAggregator 获取全局聚合器
func GetGlobalAggregator() *LogAggregator {
	aggregatorMutex.Lock()
	defer aggregatorMutex.Unlock()
	return globalAggregator
}

// WriteToAggregator 写入日志到全局聚合器
func WriteToAggregator(entry LogEntry) error {
	aggregator := GetGlobalAggregator()
	if aggregator == nil {
		return fmt.Errorf("全局聚合器未设置")
	}
	return aggregator.WriteLog(entry)
}

// 扩展logrus的Hook来支持聚合
type AggregatorHook struct {
	aggregator *LogAggregator
	service    string
}

// NewAggregatorHook 创建新的聚合器Hook
func NewAggregatorHook(aggregator *LogAggregator, service string) *AggregatorHook {
	return &AggregatorHook{
		aggregator: aggregator,
		service:    service,
	}
}

// Levels 返回支持的日志级别
func (h *AggregatorHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire 处理日志条目
func (h *AggregatorHook) Fire(entry *logrus.Entry) error {
	logEntry := LogEntry{
		Timestamp: entry.Time.Format(time.RFC3339),
		Level:     entry.Level.String(),
		Message:   entry.Message,
		Service:   h.service,
		Fields:    make(map[string]any),
	}

	// 提取TraceID和SpanID
	if traceID, ok := entry.Data["trace_id"].(string); ok {
		logEntry.TraceID = traceID
	}
	if spanID, ok := entry.Data["span_id"].(string); ok {
		logEntry.SpanID = spanID
	}

	// 提取调用者信息
	if entry.Caller != nil {
		logEntry.Caller = fmt.Sprintf("%s:%d", filepath.Base(entry.Caller.File), entry.Caller.Line)
	}

	// 复制其他字段
	for key, value := range entry.Data {
		if key != "trace_id" && key != "span_id" {
			logEntry.Fields[key] = value
		}
	}

	return h.aggregator.WriteLog(logEntry)
}
