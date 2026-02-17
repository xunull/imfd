package geo

// defaultChinaCities 中国主要城市坐标数据（省会及主要城市）
// 用于离线 GPS 反查
var defaultChinaCities = []ChinaCity{
	// 直辖市
	{Name: "北京", Province: "北京", Latitude: 39.9042, Longitude: 116.4074},
	{Name: "天津", Province: "天津", Latitude: 39.0842, Longitude: 117.2010},
	{Name: "上海", Province: "上海", Latitude: 31.2304, Longitude: 121.4737},
	{Name: "重庆", Province: "重庆", Latitude: 29.5630, Longitude: 106.5516},

	// 河北省
	{Name: "石家庄", Province: "河北", Latitude: 38.0428, Longitude: 114.5149},
	{Name: "唐山", Province: "河北", Latitude: 39.6309, Longitude: 118.1802},
	{Name: "秦皇岛", Province: "河北", Latitude: 39.9354, Longitude: 119.5977},
	{Name: "邯郸", Province: "河北", Latitude: 36.6257, Longitude: 114.5391},
	{Name: "保定", Province: "河北", Latitude: 38.8741, Longitude: 115.4646},
	{Name: "张家口", Province: "河北", Latitude: 40.7676, Longitude: 114.8868},
	{Name: "承德", Province: "河北", Latitude: 40.9510, Longitude: 117.9634},
	{Name: "廊坊", Province: "河北", Latitude: 39.5380, Longitude: 116.6839},

	// 山西省
	{Name: "太原", Province: "山西", Latitude: 37.8706, Longitude: 112.5489},
	{Name: "大同", Province: "山西", Latitude: 40.0766, Longitude: 113.3001},

	// 内蒙古
	{Name: "呼和浩特", Province: "内蒙古", Latitude: 40.8424, Longitude: 111.7490},
	{Name: "包头", Province: "内蒙古", Latitude: 40.6571, Longitude: 109.8400},
	{Name: "鄂尔多斯", Province: "内蒙古", Latitude: 39.6086, Longitude: 109.7812},

	// 辽宁省
	{Name: "沈阳", Province: "辽宁", Latitude: 41.8057, Longitude: 123.4315},
	{Name: "大连", Province: "辽宁", Latitude: 38.9140, Longitude: 121.6147},
	{Name: "鞍山", Province: "辽宁", Latitude: 41.1087, Longitude: 122.9956},

	// 吉林省
	{Name: "长春", Province: "吉林", Latitude: 43.8171, Longitude: 125.3235},
	{Name: "吉林", Province: "吉林", Latitude: 43.8380, Longitude: 126.5495},

	// 黑龙江省
	{Name: "哈尔滨", Province: "黑龙江", Latitude: 45.8038, Longitude: 126.5350},
	{Name: "齐齐哈尔", Province: "黑龙江", Latitude: 47.3542, Longitude: 123.9180},

	// 江苏省
	{Name: "南京", Province: "江苏", Latitude: 32.0603, Longitude: 118.7969},
	{Name: "苏州", Province: "江苏", Latitude: 31.2990, Longitude: 120.5853},
	{Name: "无锡", Province: "江苏", Latitude: 31.5689, Longitude: 120.2886},
	{Name: "常州", Province: "江苏", Latitude: 31.8106, Longitude: 119.9741},
	{Name: "南通", Province: "江苏", Latitude: 31.9800, Longitude: 120.8943},
	{Name: "扬州", Province: "江苏", Latitude: 32.3942, Longitude: 119.4126},
	{Name: "徐州", Province: "江苏", Latitude: 34.2055, Longitude: 117.2844},

	// 浙江省
	{Name: "杭州", Province: "浙江", Latitude: 30.2741, Longitude: 120.1551},
	{Name: "宁波", Province: "浙江", Latitude: 29.8683, Longitude: 121.5440},
	{Name: "温州", Province: "浙江", Latitude: 27.9939, Longitude: 120.6993},
	{Name: "嘉兴", Province: "浙江", Latitude: 30.7465, Longitude: 120.7555},
	{Name: "金华", Province: "浙江", Latitude: 29.0787, Longitude: 119.6495},

	// 安徽省
	{Name: "合肥", Province: "安徽", Latitude: 31.8206, Longitude: 117.2272},
	{Name: "芜湖", Province: "安徽", Latitude: 31.3340, Longitude: 118.3761},

	// 福建省
	{Name: "福州", Province: "福建", Latitude: 26.0745, Longitude: 119.2965},
	{Name: "厦门", Province: "福建", Latitude: 24.4798, Longitude: 118.0894},
	{Name: "泉州", Province: "福建", Latitude: 24.8741, Longitude: 118.6757},

	// 江西省
	{Name: "南昌", Province: "江西", Latitude: 28.6820, Longitude: 115.8579},

	// 山东省
	{Name: "济南", Province: "山东", Latitude: 36.6512, Longitude: 117.1200},
	{Name: "青岛", Province: "山东", Latitude: 36.0671, Longitude: 120.3826},
	{Name: "烟台", Province: "山东", Latitude: 37.4638, Longitude: 121.4479},
	{Name: "威海", Province: "山东", Latitude: 37.5091, Longitude: 122.1209},
	{Name: "潍坊", Province: "山东", Latitude: 36.7068, Longitude: 119.1618},

	// 河南省
	{Name: "郑州", Province: "河南", Latitude: 34.7466, Longitude: 113.6253},
	{Name: "洛阳", Province: "河南", Latitude: 34.6190, Longitude: 112.4540},
	{Name: "开封", Province: "河南", Latitude: 34.7979, Longitude: 114.3072},

	// 湖北省
	{Name: "武汉", Province: "湖北", Latitude: 30.5928, Longitude: 114.3055},
	{Name: "宜昌", Province: "湖北", Latitude: 30.6918, Longitude: 111.2867},

	// 湖南省
	{Name: "长沙", Province: "湖南", Latitude: 28.2280, Longitude: 112.9388},
	{Name: "株洲", Province: "湖南", Latitude: 27.8274, Longitude: 113.1340},
	{Name: "张家界", Province: "湖南", Latitude: 29.1170, Longitude: 110.4793},

	// 广东省
	{Name: "广州", Province: "广东", Latitude: 23.1291, Longitude: 113.2644},
	{Name: "深圳", Province: "广东", Latitude: 22.5431, Longitude: 114.0579},
	{Name: "珠海", Province: "广东", Latitude: 22.2710, Longitude: 113.5767},
	{Name: "佛山", Province: "广东", Latitude: 23.0218, Longitude: 113.1218},
	{Name: "东莞", Province: "广东", Latitude: 23.0209, Longitude: 113.7518},
	{Name: "惠州", Province: "广东", Latitude: 23.1116, Longitude: 114.4161},
	{Name: "中山", Province: "广东", Latitude: 22.5176, Longitude: 113.3926},
	{Name: "汕头", Province: "广东", Latitude: 23.3537, Longitude: 116.6819},

	// 广西
	{Name: "南宁", Province: "广西", Latitude: 22.8170, Longitude: 108.3665},
	{Name: "桂林", Province: "广西", Latitude: 25.2740, Longitude: 110.2992},

	// 海南省
	{Name: "海口", Province: "海南", Latitude: 20.0440, Longitude: 110.1999},
	{Name: "三亚", Province: "海南", Latitude: 18.2528, Longitude: 109.5120},

	// 四川省
	{Name: "成都", Province: "四川", Latitude: 30.5723, Longitude: 104.0665},
	{Name: "绵阳", Province: "四川", Latitude: 31.4670, Longitude: 104.6819},
	{Name: "乐山", Province: "四川", Latitude: 29.5522, Longitude: 103.7659},

	// 贵州省
	{Name: "贵阳", Province: "贵州", Latitude: 26.6470, Longitude: 106.6302},

	// 云南省
	{Name: "昆明", Province: "云南", Latitude: 25.0389, Longitude: 102.7183},
	{Name: "大理", Province: "云南", Latitude: 25.6065, Longitude: 100.2676},
	{Name: "丽江", Province: "云南", Latitude: 26.8721, Longitude: 100.2300},

	// 西藏
	{Name: "拉萨", Province: "西藏", Latitude: 29.6500, Longitude: 91.1100},

	// 陕西省
	{Name: "西安", Province: "陕西", Latitude: 34.3416, Longitude: 108.9398},

	// 甘肃省
	{Name: "兰州", Province: "甘肃", Latitude: 36.0611, Longitude: 103.8343},
	{Name: "敦煌", Province: "甘肃", Latitude: 40.1421, Longitude: 94.6618},

	// 青海省
	{Name: "西宁", Province: "青海", Latitude: 36.6171, Longitude: 101.7782},

	// 宁夏
	{Name: "银川", Province: "宁夏", Latitude: 38.4872, Longitude: 106.2309},

	// 新疆
	{Name: "乌鲁木齐", Province: "新疆", Latitude: 43.8256, Longitude: 87.6168},
	{Name: "喀什", Province: "新疆", Latitude: 39.4704, Longitude: 75.9893},

	// 特别行政区
	{Name: "香港", Province: "香港", Latitude: 22.3193, Longitude: 114.1694},
	{Name: "澳门", Province: "澳门", Latitude: 22.1987, Longitude: 113.5439},

	// 台湾
	{Name: "台北", Province: "台湾", Latitude: 25.0330, Longitude: 121.5654},
	{Name: "高雄", Province: "台湾", Latitude: 22.6273, Longitude: 120.3014},
}
