package handler

import (
	"MediaWarp/internal/logging"
	"MediaWarp/internal/service/jellyfin"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

// 修改播放信息请求
//
// /Items/:itemId
// 强制将 HTTPStrm 设置为支持直链播放和转码、AlistStrm 设置为支持直链播放并且禁止转码
func (jellyfinHandler *JellyfinHandler) ModifyPlaybackInfoWing(rw *http.Response) error {
	defer rw.Body.Close()
	data, err := readBody(rw)
	if err != nil {
		logging.Warning("读取响应体失败：", err)
		return err
	}

	var playbackInfoResponse jellyfin.PlaybackInfoResponse
	if err = json.Unmarshal(data, &playbackInfoResponse); err != nil {
		logging.Warning("解析 jellyfin.PlaybackInfoResponse JSON 错误：", err)
		return err
	}
	// 不行，没法播放
	if strings.Contains(rw.Request.Header.Get("Authorization"), "Findroid") {
		// Findroid处理
		if len(playbackInfoResponse.MediaSources) > 1 {
			// 对 MediaSources 排序，将Protocol=="Http"的排在最前面
			sort.Slice(playbackInfoResponse.MediaSources, func(i, j int) bool {
				protocolI := playbackInfoResponse.MediaSources[i].Protocol
				protocolJ := playbackInfoResponse.MediaSources[j].Protocol

				isHttpI := protocolI != nil && *protocolI == "Http"
				isHttpJ := protocolJ != nil && *protocolJ == "Http"

				// Http协议的排在前面
				if isHttpI && !isHttpJ {
					return true
				}
				if !isHttpI && isHttpJ {
					return false
				}
				// 如果都是Http或都不是Http，保持原有顺序
				return false
			})
			if playbackInfoResponse.MediaSources[0].Protocol != nil && *playbackInfoResponse.MediaSources[0].Protocol == "Http" {
				strm := playbackInfoResponse.MediaSources[0]
				logging.Infof("已重排Strm %s", *strm.Name)
				*strm.SupportsDirectPlay = true
				*strm.SupportsDirectStream = true
				//playbackInfoResponse.MediaSources = []jellyfin.MediaSourceInfo{strm}
			}
		}
	}

	//for index, mediasource := range playbackInfoResponse.MediaSources {
	//	logging.Debug("请求 ItemsServiceQueryItem：" + *mediasource.ID)
	//	itemResponse, err := jellyfinHandler.server.ItemsServiceQueryItem(*mediasource.ID, 1, "Path,MediaSources") // 查询 item 需要去除前缀仅保留数字部分
	//	if err != nil {
	//		logging.Warning("请求 ItemsServiceQueryItem 失败：", err)
	//		continue
	//	}
	//	item := itemResponse.Items[0]
	//	strmFileType, opt := recgonizeStrmFileType(*item.Path)
	//	switch strmFileType {
	//	case constants.HTTPStrm: // HTTPStrm 设置支持直链播放并且支持转码
	//		if !config.HTTPStrm.TransCode {
	//			*playbackInfoResponse.MediaSources[index].SupportsDirectPlay = true
	//			*playbackInfoResponse.MediaSources[index].SupportsDirectStream = true
	//			playbackInfoResponse.MediaSources[index].TranscodingURL = nil
	//			playbackInfoResponse.MediaSources[index].TranscodingSubProtocol = nil
	//			playbackInfoResponse.MediaSources[index].TranscodingContainer = nil
	//			if mediasource.DirectStreamURL != nil {
	//				apikeypair, err := utils.ResolveEmbyAPIKVPairs(*mediasource.DirectStreamURL)
	//				if err != nil {
	//					logging.Warning("解析API键值对失败：", err)
	//					continue
	//				}
	//				directStreamURL := fmt.Sprintf("/Videos/%s/stream?MediaSourceId=%s&Static=true&%s", *mediasource.ID, *mediasource.ID, apikeypair)
	//				playbackInfoResponse.MediaSources[index].DirectStreamURL = &directStreamURL
	//				logging.Info(*mediasource.Name, " 强制禁止转码，直链播放链接为: ", directStreamURL)
	//			}
	//		}
	//
	//	case constants.AlistStrm: // AlistStm 设置支持直链播放并且禁止转码
	//		if !config.AlistStrm.TransCode {
	//			*playbackInfoResponse.MediaSources[index].SupportsDirectPlay = true
	//			*playbackInfoResponse.MediaSources[index].SupportsDirectStream = true
	//			*playbackInfoResponse.MediaSources[index].SupportsTranscoding = false
	//			playbackInfoResponse.MediaSources[index].TranscodingURL = nil
	//			playbackInfoResponse.MediaSources[index].TranscodingSubProtocol = nil
	//			playbackInfoResponse.MediaSources[index].TranscodingContainer = nil
	//			directStreamURL := fmt.Sprintf("/Videos/%s/stream?MediaSourceId=%s&Static=true", *mediasource.ID, *mediasource.ID)
	//			if mediasource.DirectStreamURL != nil {
	//				logging.Debugf("%s 原直链播放链接： %s", *mediasource.Name, *mediasource.DirectStreamURL)
	//				apikeypair, err := utils.ResolveEmbyAPIKVPairs(*mediasource.DirectStreamURL)
	//				if err != nil {
	//					logging.Warning("解析API键值对失败：", err)
	//					continue
	//				}
	//				directStreamURL += "&" + apikeypair
	//			}
	//			playbackInfoResponse.MediaSources[index].DirectStreamURL = &directStreamURL
	//			container := strings.TrimPrefix(path.Ext(*mediasource.Path), ".")
	//			playbackInfoResponse.MediaSources[index].Container = &container
	//			logging.Infof("%s 强制禁止转码，直链播放链接为：%s，容器为： %s", *mediasource.Name, directStreamURL, container)
	//		} else {
	//			logging.Infof("%s 保持原有转码设置", *mediasource.Name)
	//		}
	//
	//		if playbackInfoResponse.MediaSources[index].Size == nil {
	//			alistServer, err := service.GetAlistServer(opt.(string))
	//			if err != nil {
	//				logging.Warning("获取 AlistServer 失败：", err)
	//				continue
	//			}
	//			fsGetData, err := alistServer.FsGet(*mediasource.Path)
	//			if err != nil {
	//				logging.Warning("请求 FsGet 失败：", err)
	//				continue
	//			}
	//			playbackInfoResponse.MediaSources[index].Size = &fsGetData.Size
	//			logging.Infof("%s 设置文件大小为：%d", *mediasource.Name, fsGetData.Size)
	//		}
	//	}
	//}

	if data, err = json.Marshal(playbackInfoResponse); err != nil {
		logging.Warning("序列化 jellyfin.PlaybackInfoResponse Json 错误：", err)
		return err
	}

	rw.Header.Set("Content-Type", "application/json") // 更新 Content-Type 头
	return updateBody(rw, data)
}
