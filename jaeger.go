package trace

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

// JaegerConfig 配置结构
type JaegerConfig struct {
	Endpoint    string
	ServiceName string
	Environment string
	Version     string
	Enabled     bool
}

// DefaultJaegerConfig 默认配置
func DefaultJaegerConfig() *JaegerConfig {
	return &JaegerConfig{
		Endpoint:    "http://localhost:4318/v1/traces", // 使用标准OTLP HTTP端点
		ServiceName: "trace-service",
		Environment: "development",
		Version:     "1.0.0",
		Enabled:     true,
	}
}

// LoadJaegerConfigFromEnv 从环境变量加载配置
func LoadJaegerConfigFromEnv() *JaegerConfig {
	config := DefaultJaegerConfig()

	// 支持多种环境变量名称
	if endpoint := getFirstEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "JAEGER_ENDPOINT"); endpoint != "" {
		config.Endpoint = endpoint
	}

	if serviceName := getFirstEnv("OTEL_SERVICE_NAME", "JAEGER_SERVICE_NAME"); serviceName != "" {
		config.ServiceName = serviceName
	}

	if env := getFirstEnv("OTEL_RESOURCE_ATTRIBUTES_DEPLOYMENT_ENVIRONMENT", "JAEGER_ENVIRONMENT"); env != "" {
		config.Environment = env
	}

	if version := getFirstEnv("OTEL_SERVICE_VERSION", "JAEGER_VERSION"); version != "" {
		config.Version = version
	}

	if enabled := getFirstEnv("OTEL_TRACES_EXPORTER", "JAEGER_ENABLED"); enabled != "" {
		if enabled == "otlp" || enabled == "jaeger" {
			config.Enabled = true
		} else if parsed, err := strconv.ParseBool(enabled); err == nil {
			config.Enabled = parsed
		}
	}

	return config
}

// InitJaeger 初始化Jaeger追踪
func InitJaeger(config *JaegerConfig) (func(), error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if !config.Enabled {
		return func() {}, nil
	}

	// 验证配置
	if err := validateJaegerConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建OTLP HTTP exporter
	exporter, err := createOTLPExporter(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// 创建资源
	res, err := createResource(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(createSampler(config)),
	)

	// 设置全局trace provider
	otel.SetTracerProvider(tp)

	// 设置全局propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 返回清理函数
	return func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Error shutting down tracer provider: %v\n", err)
		}
	}, nil
}

// GetTracer 获取tracer实例
func GetTracer(instrumentationName string) trace.Tracer {
	if instrumentationName == "" {
		instrumentationName = "github.com/HsiaoL1/trace"
	}
	return otel.Tracer(instrumentationName)
}

// getFirstEnv 获取第一个不为空的环境变量
func getFirstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

// validateJaegerConfig 验证Jaeger配置
func validateJaegerConfig(config *JaegerConfig) error {
	if config.ServiceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if config.Endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}
	return nil
}

// createOTLPExporter 创建OTLP导出器
func createOTLPExporter(ctx context.Context, config *JaegerConfig) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(config.Endpoint),
	}

	// 根据端点协议决定是否使用TLS
	if strings.HasPrefix(config.Endpoint, "http://") {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	return otlptracehttp.New(ctx, opts...)
}

// createResource 创建资源
func createResource(ctx context.Context, config *JaegerConfig) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.Version),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
}

// createSampler 创建采样器
func createSampler(config *JaegerConfig) sdktrace.Sampler {
	// 在开发环境中使用全量采样，生产环境中使用概率采样
	if config.Environment == "development" || config.Environment == "dev" {
		return sdktrace.AlwaysSample()
	}
	return sdktrace.TraceIDRatioBased(0.1) // 10%采样率
}
