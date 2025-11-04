package handler

import (
	"MediaWarp/internal/logging"
	"MediaWarp/internal/service/jellyfin"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// 修改播放信息请求
//
// 参考 func (jellyfinHandler *JellyfinHandler) ModifyPlaybackInfo
func (jellyfinHandler *JellyfinHandler) ModifyPlaybackInfoWing(rw *http.Response) error {
	defer rw.Body.Close()
	data, err := io.ReadAll(rw.Body)
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
				*strm.SupportsTranscoding = false
				strm.TranscodingURL = nil
				strm.TranscodingSubProtocol = nil
				strm.TranscodingContainer = nil
				//playbackInfoResponse.MediaSources = []jellyfin.MediaSourceInfo{strm}
			}
		}
	}

	if data, err = json.Marshal(playbackInfoResponse); err != nil {
		logging.Warning("序列化 jellyfin.PlaybackInfoResponse Json 错误：", err)
		return err
	}

	rw.Header.Set("Content-Type", "application/json") // 更新 Content-Type 头
	rw.Header.Set("Content-Length", strconv.Itoa(len(data)))
	rw.Body = io.NopCloser(bytes.NewReader(data))
	return nil
}
