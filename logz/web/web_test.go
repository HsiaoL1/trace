package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWebServer(t *testing.T) {
	// 创建临时日志目录
	tempDir := t.TempDir()

	// 创建测试日志文件
	testLogFile := filepath.Join(tempDir, "test.log")
	err := os.WriteFile(testLogFile, []byte(`{"timestamp":"2024-01-15T10:30:00Z","level":"info","msg":"test message","service":"test-service"}`), 0644)
	if err != nil {
		t.Fatalf("创建测试日志文件失败: %v", err)
	}

	// 创建Web服务器
	server := NewWebServer(tempDir, "8080")

	// 测试获取文件列表
	t.Run("GetLogFiles", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/files", nil)
		w := httptest.NewRecorder()
		server.getLogFiles(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200，得到 %d", w.Code)
		}

		var response LogViewResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if !response.Success {
			t.Errorf("期望成功响应，得到错误: %s", response.Error)
		}

		// 使用类型断言处理interface{}类型
		filesData, ok := response.Data.([]interface{})
		if !ok {
			t.Fatal("响应数据类型错误")
		}

		if len(filesData) == 0 {
			t.Error("期望至少有一个日志文件")
		}

		found := false
		for _, fileData := range filesData {
			fileMap, ok := fileData.(map[string]interface{})
			if !ok {
				continue
			}
			if name, ok := fileMap["name"].(string); ok && name == "test.log" {
				found = true
				break
			}
		}

		if !found {
			t.Error("未找到测试日志文件")
		}
	})

	// 测试获取日志内容
	t.Run("GetLogContent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/files/content/test.log", nil)
		w := httptest.NewRecorder()
		server.getLogContent(w, req, "test.log")

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200，得到 %d", w.Code)
		}

		var response LogViewResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if !response.Success {
			t.Errorf("期望成功响应，得到错误: %s", response.Error)
		}

		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("响应数据类型错误")
		}

		content, ok := data["content"].([]interface{})
		if !ok {
			t.Fatal("内容数据类型错误")
		}

		if len(content) == 0 {
			t.Error("期望有日志内容")
		}
	})

	// 测试获取统计信息
	t.Run("GetLogStats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/stats", nil)
		w := httptest.NewRecorder()
		server.getLogStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200，得到 %d", w.Code)
		}

		var response LogViewResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if !response.Success {
			t.Errorf("期望成功响应，得到错误: %s", response.Error)
		}

		stats, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("统计数据类型错误")
		}

		totalFiles, ok := stats["total_files"].(float64) // JSON数字默认解析为float64
		if !ok {
			t.Fatal("总文件数类型错误")
		}

		if totalFiles == 0 {
			t.Error("期望至少有一个日志文件")
		}
	})

	// 测试删除文件
	t.Run("DeleteLogFile", func(t *testing.T) {
		// 创建要删除的测试文件
		deleteFile := filepath.Join(tempDir, "delete_test.log")
		err := os.WriteFile(deleteFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("创建删除测试文件失败: %v", err)
		}

		req := httptest.NewRequest("DELETE", "/api/files/delete/delete_test.log", nil)
		w := httptest.NewRecorder()
		server.deleteLogFile(w, req, "delete_test.log")

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200，得到 %d", w.Code)
		}

		var response LogViewResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if !response.Success {
			t.Errorf("期望成功响应，得到错误: %s", response.Error)
		}

		// 验证文件已被删除
		if _, err := os.Stat(deleteFile); !os.IsNotExist(err) {
			t.Error("文件应该已被删除")
		}
	})

	// 测试安全防护
	t.Run("SecurityPathTraversal", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/files/content/../../../etc/passwd", nil)
		w := httptest.NewRecorder()
		server.getLogContent(w, req, "../../../etc/passwd")

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200，得到 %d", w.Code)
		}

		var response LogViewResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if response.Success {
			t.Error("期望失败响应，因为路径遍历攻击")
		}

		if response.Error == "" {
			t.Error("期望有错误消息")
		}
	})
}

func TestFileInfo(t *testing.T) {
	// 测试文件信息结构
	fileInfo := FileInfo{
		Name:         "test.log",
		Size:         1024,
		ModTime:      time.Now(),
		IsCompressed: false,
	}

	if fileInfo.Name != "test.log" {
		t.Errorf("期望文件名 test.log，得到 %s", fileInfo.Name)
	}

	if fileInfo.Size != 1024 {
		t.Errorf("期望文件大小 1024，得到 %d", fileInfo.Size)
	}

	if fileInfo.IsCompressed {
		t.Error("期望文件未压缩")
	}
}

func TestLogViewResponse(t *testing.T) {
	// 测试响应结构
	response := LogViewResponse{
		Success: true,
		Data:    "test data",
		Error:   "",
	}

	if !response.Success {
		t.Error("期望成功响应")
	}

	if response.Data != "test data" {
		t.Errorf("期望数据 test data，得到 %v", response.Data)
	}

	if response.Error != "" {
		t.Errorf("期望无错误，得到 %s", response.Error)
	}
}

func BenchmarkGetLogFiles(b *testing.B) {
	// 创建临时目录和测试文件
	tempDir := b.TempDir()
	for i := 0; i < 100; i++ {
		filename := fmt.Sprintf("test_%d.log", i)
		filepath := filepath.Join(tempDir, filename)
		os.WriteFile(filepath, []byte("test content"), 0644)
	}

	server := NewWebServer(tempDir, "8080")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/files", nil)
		w := httptest.NewRecorder()
		server.getLogFiles(w, req)
	}
}
