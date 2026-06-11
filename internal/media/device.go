package media

import "strings"

// device_type derived 字段。
//
// EXIF 的 CameraMake 字段告诉我们品牌，但用户想区分「手机 vs 相机」。
// 内置一个 lowercase + substring 的映射表：v1 写死 Go 代码，覆盖 demo 必需 + 主流品牌。
// 未来如果 unknown 太多再考虑外置 YAML (LIST-DEV-1)。

const (
	DeviceTypePhone   = "phone"
	DeviceTypeCamera  = "camera"
	DeviceTypeUnknown = "unknown"
)

// 全部小写。substring match：CameraMake 经过 lowercase 后 contains 任一关键字即归该类。
var phoneMakes = []string{
	"apple",
	"samsung",
	"xiaomi",
	"redmi",
	"huawei",
	"honor",
	"oppo",
	"vivo",
	"oneplus",
	"google",
	"realme",
	"sony mobile",
	"asus rog",
}

var cameraMakes = []string{
	"sony",
	"nikon",
	"canon",
	"fujifilm",
	"olympus",
	"panasonic",
	"leica",
	"pentax",
	"hasselblad",
	"ricoh",
}

// DeviceType 根据 EXIF CameraMake 字段判定设备类型。
//
//   "Apple"               → phone
//   "Apple iPhone"        → phone
//   "NIKON CORPORATION"   → camera
//   "Sony"                → camera   注意：Sony Mobile 才是 phone（substring 优先级看顺序）
//   "Sony Mobile"         → phone    phone 表先扫
//   "DJI"                 → unknown
//   ""                    → unknown
//
// 注：phone 表先扫，因为 "Sony Mobile" 既含 "sony" 又含 "sony mobile"，
// 应当判为 phone。这也是为什么 phone 表里有更具体的 "sony mobile"。
func DeviceType(make string) string {
	if make == "" {
		return DeviceTypeUnknown
	}
	low := strings.ToLower(make)
	for _, kw := range phoneMakes {
		if strings.Contains(low, kw) {
			return DeviceTypePhone
		}
	}
	for _, kw := range cameraMakes {
		if strings.Contains(low, kw) {
			return DeviceTypeCamera
		}
	}
	return DeviceTypeUnknown
}
