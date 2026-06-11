package media

import "testing"

func TestDeviceType(t *testing.T) {
	cases := []struct {
		make string
		want string
	}{
		// phone
		{"Apple", DeviceTypePhone},
		{"apple", DeviceTypePhone},
		{"Samsung", DeviceTypePhone},
		{"Xiaomi", DeviceTypePhone},
		{"Redmi", DeviceTypePhone},
		{"Huawei", DeviceTypePhone},
		{"HONOR", DeviceTypePhone},
		{"OPPO", DeviceTypePhone},
		{"vivo", DeviceTypePhone},
		{"OnePlus", DeviceTypePhone},
		{"Google", DeviceTypePhone},
		{"realme", DeviceTypePhone},
		{"Sony Mobile", DeviceTypePhone}, // 优先匹配 phone 表
		// camera
		{"Sony", DeviceTypeCamera},
		{"SONY", DeviceTypeCamera},
		{"Nikon", DeviceTypeCamera},
		{"NIKON CORPORATION", DeviceTypeCamera}, // 大写 + 公司后缀
		{"Canon", DeviceTypeCamera},
		{"Fujifilm", DeviceTypeCamera},
		{"FUJIFILM", DeviceTypeCamera},
		{"OLYMPUS", DeviceTypeCamera},
		{"Panasonic", DeviceTypeCamera},
		{"Leica Camera AG", DeviceTypeCamera},
		{"PENTAX", DeviceTypeCamera},
		{"Hasselblad", DeviceTypeCamera},
		{"RICOH IMAGING COMPANY, LTD.", DeviceTypeCamera},
		// unknown
		{"DJI", DeviceTypeUnknown},
		{"GoPro", DeviceTypeUnknown},
		{"", DeviceTypeUnknown},
		{"unknown brand", DeviceTypeUnknown},
	}
	for _, c := range cases {
		got := DeviceType(c.make)
		if got != c.want {
			t.Errorf("DeviceType(%q) = %q, want %q", c.make, got, c.want)
		}
	}
}
