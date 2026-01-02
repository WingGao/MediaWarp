package handler

import (
	"MediaWarp/internal/config"
	"MediaWarp/internal/logging"
	"MediaWarp/internal/service"
	"MediaWarp/internal/service/alist"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/allegro/bigcache/v3"
)

type StrmHandlerFunc func(content string, ua string) string

func getHTTPStrmHandler() (StrmHandlerFunc, error) {
	var cache *bigcache.BigCache
	if config.Cache.Enable && config.Cache.HTTPStrmTTL > 0 && config.HTTPStrm.FinalURL {
		var err error
		cache, err = bigcache.New(context.Background(), bigcache.DefaultConfig(config.Cache.HTTPStrmTTL))
		if err != nil {
			return nil, fmt.Errorf("创建 HTTPStrm 缓存失败: %w", err)
		}
		logging.Info("启用 HTTPStrm 缓存，TTL: ", config.Cache.HTTPStrmTTL)
	}

	client := &http.Client{ // 创建自定义HTTP客户端配置
		Timeout: RedirectTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 禁止自动重定向，以便手动处理
			return http.ErrUseLastResponse
		},
	}
	return func(content string, ua string) string {
		if config.HTTPStrm.FinalURL {
			if cache != nil {
				if cachedURL, err := cache.Get(content); err == nil {
					logging.Infof("HTTPStrm 重定向至: %s (缓存)", string(cachedURL))
					return string(cachedURL)
				}
			}

			logging.Debug("HTTPStrm 启用获取最终 URL，开始尝试获取最终 URL")
			finalURL, err := getFinalURL(client, content, ua)
			if err != nil {
				logging.Warning("获取最终 URL 失败，使用原始 URL: ", err)
			} else {
				logging.Info("HTTPStrm 重定向至: ", finalURL)
			}
			if cache != nil {
				if err := cache.Set(content, []byte(finalURL)); err != nil {
					logging.Warning("缓存 HTTPStrm URL 失败: ", err)
				} else {
					logging.Debug("缓存 HTTPStrm URL 成功")
				}
			}
			return finalURL
		} else {
			logging.Debug("HTTPStrm 未启用获取最终 URL，直接使用原始 URL: ", content)
			return content
		}
	}, nil
}

func alistStrmHandler(content string, alistConfig config.AlistSetting) string {
	alistClient, err := service.GetAlistClient(alistConfig.ADDR)
	if err != nil {
		logging.Warning("获取 AlistClient 失败：", err)
		return ""
	}
	url, err := alistClient.GetFileURL(content, config.AlistStrm.RawURL)
	if err != nil {
		logging.Warning("获取文件 URL 失败：", err)
		return ""
	}
	logging.Infof("AlistStrm 重定向至：%s", url)
	return url
}

func alistFileHandler(content string, alistConfig config.AlistSetting) string {
	alistClient, err := service.GetAlistClient(alistConfig.ADDR)
	if err != nil {
		logging.Warning("获取 AlistClient 失败：", err)
		return ""
	}
	localPath := content

	// 尝试判断localPath是否是符号链接，并返回链接的目标路径
	realPath := localPath
	if info, err := os.Lstat(localPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Readlink(localPath); err == nil {
			// 如果是相对路径，转换为绝对路径
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(localPath), target)
			}
			realPath = filepath.Clean(target)
			logging.Debugf("符号链接解析：%s -> %s", localPath, realPath)
		} else {
			logging.Debugf("读取符号链接失败：%s, 错误：%v", localPath, err)
		}
	}

	// 判断localPath是否在 PathMapper 中, 并转换为 alistPath
	alistPath := ""
	if alistConfig.PathMapper != nil {
		for localPrefix, alistPrefix := range alistConfig.PathMapper {
			if strings.HasPrefix(realPath, localPrefix) {
				alistPath = strings.Replace(realPath, localPrefix, alistPrefix, 1)
				logging.Debugf("路径映射：%s -> %s", realPath, alistPath)
				break
			}
		}
	}

	// 如果没有匹配的路径映射，使用原始内容
	if alistPath == "" {
		alistPath = content
	}

	// 判断alistPath是否真实存在 FsGet
	_, err = alistClient.FsGet(&alist.FsGetRequest{Path: alistPath, Page: 1})
	if err != nil {
		logging.Warningf("文件不存在于 Alist：%s，错误：%v", alistPath, err)
		return ""
	}

	// 如果存在，返回 GetFileURL
	url, err := alistClient.GetFileURL(alistPath, config.AlistStrm.RawURL)
	if err != nil {
		logging.Warning("获取文件 URL 失败：", err)
		return ""
	}
	logging.Infof("AlistStrm 重定向至：%s", url)
	return url
}
