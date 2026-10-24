// Package protocol: SmsForwarder 3.3.x 线上协议的数据结构与信封。
// 字段名与上游 Gson @SerializedName 逐一对应（SmsForwarder/app/.../server/model）。
package protocol

import "encoding/json"

// SafetyMode 上游安全措施：0=无 1=时间戳签名 2=RSA 3=SM4
type SafetyMode int

const (
	SafetyNone SafetyMode = iota
	SafetySign
	SafetyRSA
	SafetySM4
)

// BaseRequest 上游请求信封（ClientFragment.queryConfig 的 msgMap）
type BaseRequest struct {
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
	Sign      string          `json:"sign,omitempty"`
}

// BaseResponse 上游统一响应（HttpServerUtils.response）：HTTP 200 不代表成功，必须检查 code
type BaseResponse struct {
	Code      int             `json:"code"` // 200 成功；500 业务失败
	Msg       string          `json:"msg"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
	Sign      string          `json:"sign,omitempty"`
}

// SimInfo 上游 entity/SimInfo（gson Map<Int,SimInfo>，键为字符串）
type SimInfo struct {
	MCarrierName   string `json:"mCarrierName"`
	MIccId         string `json:"mIccId"`
	MSimSlotIndex  int    `json:"mSimSlotIndex"`
	MNumber        string `json:"mNumber"`
	MCountryIso    string `json:"mCountryIso"`
	MSubscription  int    `json:"mSubscriptionId"`
}

// ConfigData /config/query 返回；九个 enable_api_* 决定操作入口
type ConfigData struct {
	EnableApiClone        bool              `json:"enable_api_clone"`
	EnableApiSmsSend      bool              `json:"enable_api_sms_send"`
	EnableApiSmsQuery     bool              `json:"enable_api_sms_query"`
	EnableApiCallQuery    bool              `json:"enable_api_call_query"`
	EnableApiContactQuery bool              `json:"enable_api_contact_query"`
	EnableApiContactAdd   bool              `json:"enable_api_contact_add"`
	EnableApiBatteryQuery bool              `json:"enable_api_battery_query"`
	EnableApiWol          bool              `json:"enable_api_wol"`
	EnableApiLocation     bool              `json:"enable_api_location"`
	ExtraDeviceMark       string            `json:"extra_device_mark"`
	ExtraSim1             string            `json:"extra_sim1"`
	ExtraSim2             string            `json:"extra_sim2"`
	SimInfoList           map[string]SimInfo `json:"sim_info_list"`
	VersionCode           int               `json:"version_code"`
	VersionName           string            `json:"version_name"`
}

// SmsInfo 短信记录
type SmsInfo struct {
	Name    string `json:"name"`
	Number  string `json:"number"`
	Content string `json:"content"`
	Date    int64  `json:"date"`
	Type    int    `json:"type"`            // 1=接收 2=发送
	SimID   int    `json:"sim_id"`          // 0=SIM1 1=SIM2 -1=获取失败
	SubID   int    `json:"sub_id"`
}

// CallInfo 通话记录（真机字段：via_number/sim_id/sub_id/is_forwarded）
type CallInfo struct {
	Name        string `json:"name"`
	Number      string `json:"number"`
	DateLong    int64  `json:"dateLong"`
	Duration    int    `json:"duration"`
	Type        int    `json:"type"` // 1=呼入 2=呼出 3=未接
	ViaNumber   string `json:"via_number"`
	SimID       int    `json:"sim_id"`
	SubID       int    `json:"sub_id"`
	IsForwarded bool   `json:"is_forwarded"`
}

// ContactInfo 联系人（真机字段 phone_number；firstLetter 为上游计算属性不序列化）
type ContactInfo struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
}

// BatteryInfo 电量（上游字段全部为字符串）
type BatteryInfo struct {
	Level       string `json:"level"`
	Scale       string `json:"scale"`
	Voltage     string `json:"voltage"`
	Temperature string `json:"temperature"`
	Status      string `json:"status"`
	Health      string `json:"health"`
	Plugged     string `json:"plugged"`
	Msg         string `json:"msg,omitempty"`
}

// LocationInfo 定位缓存（time 为定位发生时间）
type LocationInfo struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Address   string  `json:"address"`
	Time      string  `json:"time"`
	Provider  string  `json:"provider"`
	Msg       string  `json:"msg,omitempty"`
}
