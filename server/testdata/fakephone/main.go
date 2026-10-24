// fakephone: 测试用假 SmsForwarder 手机（协议语义对齐上游 3.3.x）。
// 用法：fakephone [-addr :15000] [-mode none|sign|sm4] [-sign-key KEY] [-sm4-key HEX32]
// 验证边界（开发计划 §7）：模拟测试验证协议格式，不代表真机验收。
package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"smsfm/server/internal/protocol"
)

func main() {
	addr := flag.String("addr", ":15000", "监听地址")
	mode := flag.Int("mode", 0, "安全模式 0/1/3（RSA 见单元测试）")
	signKey := flag.String("sign-key", "fake-sign-key", "签名密钥")
	sm4Key := flag.String("sm4-key", "00112233445566778899AABBCCDDEEFF", "SM4 密钥（32 位十六进制）")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		envJSON, err := open(body, *mode, *signKey, *sm4Key)
		if err != nil {
			write(w, *mode, *sm4Key, map[string]any{"code": 500, "msg": "请求解密失败：" + err.Error(), "timestamp": time.Now().UnixMilli()})
			return
		}
		if *mode == 1 && *signKey != "" {
			var env protocol.BaseRequest
			if err := json.Unmarshal(envJSON, &env); err != nil || env.Sign == "" || env.Timestamp == 0 {
				write(w, *mode, *sm4Key, map[string]any{"code": 500, "msg": "签名校验失败：缺少 sign 或 timestamp", "timestamp": time.Now().UnixMilli()})
				return
			}
			want := protocol.CalcSign(fmt.Sprintf("%d", env.Timestamp), *signKey)
			if want != env.Sign {
				write(w, *mode, *sm4Key, map[string]any{"code": 500, "msg": "签名校验失败", "timestamp": time.Now().UnixMilli()})
				return
			}
		}

		switch r.URL.Path {
		case "/config/query":
			writeData(w, *mode, *sm4Key, map[string]any{
				"enable_api_sms_query": true, "enable_api_sms_send": true,
				"enable_api_call_query": true, "enable_api_contact_query": true, "enable_api_contact_add": true,
				"enable_api_battery_query": true, "enable_api_wol": true, "enable_api_clone": true,
				"enable_api_location":      false,
				"extra_device_mark":        "假手机-e2e",
				"extra_sim1":               "测试卡",
				"sim_info_list": map[string]any{
					"0": map[string]any{"mCarrierName": "E2E运营商", "mSimSlotIndex": 0, "mNumber": "138****0000", "mSubscriptionId": 1},
				},
				"version_code": 100053,
				"version_name": "3.3.2-fake",
			})
		case "/sms/query":
			writeData(w, *mode, *sm4Key, []map[string]any{
				{"name": "E2E测试", "number": "13800001234", "content": "【E2E】端到端验证短信 8888", "date": time.Now().UnixMilli(), "type": 1, "sim_id": 0, "sub_id": 1},
			})
		case "/sms/send":
			writeData(w, *mode, *sm4Key, "success")
		case "/battery/query":
			writeData(w, *mode, *sm4Key, map[string]any{
				"level": "77", "scale": "100", "voltage": "4050", "temperature": "29.1",
				"status": "3", "health": "2", "plugged": "0",
			})
		default:
			writeData(w, *mode, *sm4Key, map[string]any{"path": r.URL.Path})
		}
	})
	log.Printf("假手机启动 http://localhost%s（模式 %d）", *addr, *mode)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func open(body []byte, mode int, signKey, sm4Hex string) ([]byte, error) {
	switch mode {
	case 3:
		key, err := hex.DecodeString(sm4Hex)
		if err != nil {
			return nil, err
		}
		return protocol.SM4DecryptHex(string(body), key)
	default:
		return body, nil
	}
}

func write(w http.ResponseWriter, mode int, sm4Hex string, resp map[string]any) {
	raw, _ := json.Marshal(resp)
	switch mode {
	case 3:
		key, _ := hex.DecodeString(sm4Hex)
		wire, err := protocol.SM4EncryptHex(raw, key)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(wire))
	default:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(raw)
	}
	_ = base64.StdEncoding // 保持与上游对照时可读
}

func writeData(w http.ResponseWriter, mode int, sm4Hex string, data any) {
	write(w, mode, sm4Hex, map[string]any{"code": 200, "msg": "success", "data": data, "timestamp": time.Now().UnixMilli()})
}
