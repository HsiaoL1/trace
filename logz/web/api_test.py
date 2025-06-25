#!/usr/bin/env python3
"""
日志管理系统 API 测试脚本
用于测试第三方服务集成功能
"""

import requests
import json
import time
from datetime import datetime, timedelta


class LogManagerAPITest:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})

    def test_health_check(self):
        """测试健康检查"""
        print("=== 测试健康检查 ===")
        try:
            response = self.session.get(f"{self.base_url}/api/v1/health")
            print(f"状态码: {response.status_code}")
            print(f"响应: {json.dumps(response.json(), indent=2, ensure_ascii=False)}")
            return response.status_code == 200
        except Exception as e:
            print(f"健康检查失败: {e}")
            return False

    def test_write_log(self):
        """测试写入日志"""
        print("\n=== 测试写入日志 ===")
        test_data = {
            "level": "info",
            "message": "API测试 - 用户登录成功",
            "trace_id": "test_trace_123",
            "span_id": "test_span_456",
            "service": "api-test-service",
            "fields": {
                "user_id": "test_user_001",
                "ip": "192.168.1.100",
                "user_agent": "Mozilla/5.0 (Test Browser)",
            },
        }

        try:
            response = self.session.post(
                f"{self.base_url}/api/v1/logs/write", json=test_data
            )
            print(f"状态码: {response.status_code}")
            print(f"响应: {json.dumps(response.json(), indent=2, ensure_ascii=False)}")
            return response.status_code == 200
        except Exception as e:
            print(f"写入日志失败: {e}")
            return False

    def test_search_logs(self):
        """测试搜索日志"""
        print("\n=== 测试搜索日志 ===")
        search_query = {"trace_id": "test_trace_123", "limit": 10, "use_index": True}

        try:
            response = self.session.post(
                f"{self.base_url}/api/v1/logs/search", json=search_query
            )
            print(f"状态码: {response.status_code}")
            result = response.json()
            print(f"响应: {json.dumps(result, indent=2, ensure_ascii=False)}")

            if result.get("success") and result.get("data"):
                entries = result["data"].get("entries", [])
                print(f"找到 {len(entries)} 条日志记录")

            return response.status_code == 200
        except Exception as e:
            print(f"搜索日志失败: {e}")
            return False

    def test_get_errors(self):
        """测试获取错误日志"""
        print("\n=== 测试获取错误日志 ===")
        try:
            response = self.session.get(f"{self.base_url}/api/v1/logs/errors?limit=5")
            print(f"状态码: {response.status_code}")
            result = response.json()
            print(f"响应: {json.dumps(result, indent=2, ensure_ascii=False)}")
            return response.status_code == 200
        except Exception as e:
            print(f"获取错误日志失败: {e}")
            return False

    def test_get_stats(self):
        """测试获取统计信息"""
        print("\n=== 测试获取统计信息 ===")
        try:
            response = self.session.get(f"{self.base_url}/api/v1/stats")
            print(f"状态码: {response.status_code}")
            result = response.json()
            print(f"响应: {json.dumps(result, indent=2, ensure_ascii=False)}")
            return response.status_code == 200
        except Exception as e:
            print(f"获取统计信息失败: {e}")
            return False

    def test_get_files(self):
        """测试获取文件列表"""
        print("\n=== 测试获取文件列表 ===")
        try:
            response = self.session.get(f"{self.base_url}/api/v1/files")
            print(f"状态码: {response.status_code}")
            result = response.json()
            print(f"响应: {json.dumps(result, indent=2, ensure_ascii=False)}")
            return response.status_code == 200
        except Exception as e:
            print(f"获取文件列表失败: {e}")
            return False

    def test_write_multiple_logs(self):
        """测试写入多条日志"""
        print("\n=== 测试写入多条日志 ===")
        levels = ["debug", "info", "warn", "error"]
        services = [
            "user-service",
            "order-service",
            "payment-service",
            "notification-service",
        ]

        success_count = 0
        total_count = 0

        for i in range(10):
            level = levels[i % len(levels)]
            service = services[i % len(services)]

            log_data = {
                "level": level,
                "message": f"批量测试日志 #{i+1} - {level}级别",
                "trace_id": f"batch_trace_{i+1:03d}",
                "span_id": f"batch_span_{i+1:03d}",
                "service": service,
                "fields": {
                    "batch_id": f"batch_{i+1}",
                    "timestamp": datetime.now().isoformat(),
                },
            }

            try:
                response = self.session.post(
                    f"{self.base_url}/api/v1/logs/write", json=log_data
                )
                if response.status_code == 200:
                    success_count += 1
                total_count += 1
                print(
                    f"写入日志 #{i+1}: {'成功' if response.status_code == 200 else '失败'}"
                )
            except Exception as e:
                print(f"写入日志 #{i+1} 异常: {e}")
                total_count += 1

        print(f"批量写入完成: {success_count}/{total_count} 成功")
        return success_count == total_count

    def run_all_tests(self):
        """运行所有测试"""
        print("开始API测试...")
        print(f"目标服务器: {self.base_url}")
        print("=" * 50)

        tests = [
            ("健康检查", self.test_health_check),
            ("写入日志", self.test_write_log),
            ("搜索日志", self.test_search_logs),
            ("获取错误日志", self.test_get_errors),
            ("获取统计信息", self.test_get_stats),
            ("获取文件列表", self.test_get_files),
            ("批量写入日志", self.test_write_multiple_logs),
        ]

        results = []
        for test_name, test_func in tests:
            print(f"\n开始测试: {test_name}")
            try:
                result = test_func()
                results.append((test_name, result))
                print(f"测试结果: {'通过' if result else '失败'}")
            except Exception as e:
                print(f"测试异常: {e}")
                results.append((test_name, False))

        # 输出测试总结
        print("\n" + "=" * 50)
        print("测试总结:")
        passed = sum(1 for _, result in results if result)
        total = len(results)

        for test_name, result in results:
            status = "✓ 通过" if result else "✗ 失败"
            print(f"  {test_name}: {status}")

        print(f"\n总体结果: {passed}/{total} 测试通过")

        if passed == total:
            print("🎉 所有测试通过！API功能正常。")
        else:
            print("⚠️  部分测试失败，请检查服务器状态和API实现。")


def main():
    # 检查服务器是否运行
    try:
        response = requests.get("http://localhost:8080/api/v1/health", timeout=5)
        if response.status_code != 200:
            print("❌ 服务器未正常运行，请先启动日志管理Web服务器")
            print("运行命令: cd web && ./start.sh")
            return
    except requests.exceptions.RequestException:
        print("❌ 无法连接到服务器，请确保服务器正在运行")
        print("运行命令: cd web && ./start.sh")
        return

    # 运行测试
    tester = LogManagerAPITest()
    tester.run_all_tests()


if __name__ == "__main__":
    main()
