package app

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wangwei518/gin-admin/internal/app/config"
	"github.com/wangwei518/gin-admin/internal/app/initialize"
	"github.com/wangwei518/gin-admin/pkg/logger"

	// 引入swagger
	_ "github.com/wangwei518/gin-admin/internal/app/swagger"
)

type options struct {
	ConfigFile string
	ModelFile  string
	MenuFile   string
	WWWDir     string
	Version    string
}

// Option 定义配置项
type Option func(*options)

// SetConfigFile 设定配置文件
func SetConfigFile(s string) Option {
	return func(o *options) {
		o.ConfigFile = s
	}
}

// SetModelFile 设定casbin模型配置文件
func SetModelFile(s string) Option {
	return func(o *options) {
		o.ModelFile = s
	}
}

// SetWWWDir 设定静态站点目录
func SetWWWDir(s string) Option {
	return func(o *options) {
		o.WWWDir = s
	}
}

// SetMenuFile 设定菜单数据文件
func SetMenuFile(s string) Option {
	return func(o *options) {
		o.MenuFile = s
	}
}

// SetVersion 设定版本号
func SetVersion(s string) Option {
	return func(o *options) {
		o.Version = s
	}
}

// Run 运行服务
func Run(ctx context.Context, opts ...Option) error {
	var state int32 = 1
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	cleanFunc, err := Init(ctx, opts...)
	if err != nil {
		return err
	}

EXIT:
	for {
		sig := <-sc
		logger.Printf(ctx, "接收到信号[%s]", sig.String())
		switch sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			atomic.CompareAndSwapInt32(&state, 1, 0)
			break EXIT
		case syscall.SIGHUP:
		default:
			break EXIT
		}
	}

	cleanFunc()
	logger.Printf(ctx, "服务退出")
	time.Sleep(time.Second)
	os.Exit(int(atomic.LoadInt32(&state)))
	return nil
}

// Init 应用初始化
func Init(ctx context.Context, opts ...Option) (func(), error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	// 读取config文件，放入 config.C 结构中，可支持toml/yaml等多种格式
	config.MustLoad(o.ConfigFile)
	if v := o.ModelFile; v != "" {
		config.C.Casbin.Model = v
	}
	if v := o.WWWDir; v != "" {
		config.C.WWW = v
	}
	if v := o.MenuFile; v != "" {
		config.C.Menu.Data = v
	}
	
	// 初始化打印config.toml/yaml内容
	config.PrintWithJSON()
	
	// 初始化打印模式/进程号/TraceID等 
	logger.Printf(ctx, "服务启动，运行模式：%s，版本号：%s，进程号：%d", config.C.RunMode, o.Version, os.Getpid())

	// 初始化Log模块(输出方式，级别等)
	loggerCleanFunc, err := initialize.InitLogger()
	if err != nil {
		return nil, err
	}

	// 初始化服务运行监控服务 gops，配置来自 config.C.Monitor
	initialize.InitMonitor(ctx)

	// 初始化图形验证码服务, 配置来自 config.C.Captcha
	// 验证码服务来自Redis, 配置来自 config.C.Redis
	// 【内网应用环境，去除验证码功能】
	initialize.InitCaptcha()

	// 初始化依赖注入器
	injector, injectorCleanFunc, err := initialize.BuildInjector()
	if err != nil {
		return nil, err
	}

	// 初始化菜单数据
	err = injector.Menu.Load()
	if err != nil {
		return nil, err
	}

	// 初始化HTTP服务，配置来自 config.C.HTTP
	httpServerCleanFunc := initialize.InitHTTPServer(ctx, injector.Engine)

	return func() {
		// 关闭httpServer
		httpServerCleanFunc()
		// 关闭注入器
		injectorCleanFunc()
		// 关闭log模块
		loggerCleanFunc()
	}, nil
}
