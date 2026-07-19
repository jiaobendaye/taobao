// Package pizhi 皮质壳档口分配测试
package pizhi

import (
	"os"
	"path/filepath"
	"testing"
)

// 测试夹具：3 个订单 + 2 个档口配置
// 订单 1: 商品ID=988808938880, SKU=喜马拉雅白鳄鱼, 型号=iPhone15, 数量=2 -> 鹏华
// 订单 2: 商品ID=988808938880, SKU=喜马拉雅白鳄鱼, 型号=iPhone15, 数量=1 -> 鹏华（聚合）
// 订单 3: 商品ID=992724480793, SKU=奶芋紫鳄鱼纹, 型号=HuaweiPura70, 数量=3 -> 森慕为
// 订单 4: 商品ID=999999999999, SKU=未知, 型号=XX -> 未匹配
func TestProcessData_Aggregation(t *testing.T) {
	engine := &Engine{
		Items: map[string]ConfigItem{
			"988808938880|喜马拉雅白鳄鱼": {Stall: "鹏华"},
			"992724480793|奶芋紫鳄鱼纹":   {Stall: "森慕为"},
		},
		Stalls: []string{"鹏华", "森慕为"},
	}

	headers := []string{"店铺名称", "订单编号", "商品id", "商品规格", "商品数量"}
	rows := [][]string{
		{"店A", "ORD001", "988808938880", "iPhone15|喜马拉雅白鳄鱼", "2"},
		{"店A", "ORD002", "988808938880", "iPhone15|喜马拉雅白鳄鱼", "1"},
		{"店B", "ORD003", "992724480793", "HuaweiPura70|奶芋紫鳄鱼纹", "3"},
		{"店C", "ORD004", "999999999999", "XX|未知", "1"},
	}

	result := ProcessData(rows, headers, engine)

	if len(result.Unmatched) != 1 {
		t.Fatalf("未匹配数量 = %d, want 1", len(result.Unmatched))
	}
	if len(result.StallAggregates) != 2 {
		t.Fatalf("档口数 = %d, want 2", len(result.StallAggregates))
	}

	// 鹏华：型号 iPhone15, 数量 3
	penghua := result.StallAggregates["鹏华"]
	if len(penghua) != 1 {
		t.Fatalf("鹏华聚合行数 = %d, want 1", len(penghua))
	}
	if penghua[0].Model != "iPhone15" {
		t.Errorf("鹏华型号 = %s, want iPhone15", penghua[0].Model)
	}
	if penghua[0].Quantity != 3 {
		t.Errorf("鹏华数量 = %d, want 3", penghua[0].Quantity)
	}

	// 森慕为：型号 HuaweiPura70, 数量 3
	senmu := result.StallAggregates["森慕为"]
	if len(senmu) != 1 {
		t.Fatalf("森慕为聚合行数 = %d, want 1", len(senmu))
	}
	if senmu[0].Quantity != 3 {
		t.Errorf("森慕为数量 = %d, want 3", senmu[0].Quantity)
	}
}

// 不同型号下相同 (商品ID, SKU) 应该拆成多行
func TestProcessData_DifferentModelsSplit(t *testing.T) {
	engine := &Engine{
		Items: map[string]ConfigItem{
			"988808938880|喜马拉雅白鳄鱼": {Stall: "鹏华"},
		},
		Stalls: []string{"鹏华"},
	}

	headers := []string{"商品id", "商品规格", "商品数量"}
	rows := [][]string{
		{"988808938880", "iPhone15|喜马拉雅白鳄鱼", "2"},
		{"988808938880", "HuaweiPura70|喜马拉雅白鳄鱼", "1"},
	}

	result := ProcessData(rows, headers, engine)

	penghua := result.StallAggregates["鹏华"]
	if len(penghua) != 2 {
		t.Fatalf("鹏华聚合行数 = %d, want 2", len(penghua))
	}
}

// 同型号不同 SKU 应该拆成多行
func TestProcessData_DifferentSkusSplit(t *testing.T) {
	engine := &Engine{
		Items: map[string]ConfigItem{
			"988808938880|喜马拉雅白鳄鱼": {Stall: "鹏华"},
			"988808938880|烫金淡粉蛇纹":   {Stall: "鹏华"},
		},
		Stalls: []string{"鹏华"},
	}

	headers := []string{"商品id", "商品规格", "商品数量"}
	rows := [][]string{
		{"988808938880", "iPhone15|喜马拉雅白鳄鱼", "2"},
		{"988808938880", "iPhone15|烫金淡粉蛇纹", "1"},
	}

	result := ProcessData(rows, headers, engine)
	penghua := result.StallAggregates["鹏华"]
	if len(penghua) != 2 {
		t.Fatalf("鹏华聚合行数 = %d, want 2", len(penghua))
	}
}

// 空 SKU（无 | 分隔）
func TestProcessData_NoPipe(t *testing.T) {
	engine := &Engine{
		Items: map[string]ConfigItem{
			"988808938880|单品": {Stall: "鹏华"},
		},
		Stalls: []string{"鹏华"},
	}
	headers := []string{"商品id", "商品规格", "商品数量"}
	rows := [][]string{
		{"988808938880", "单品", "5"},
	}
	result := ProcessData(rows, headers, engine)
	if len(result.StallAggregates["鹏华"]) != 1 {
		t.Fatalf("鹏华聚合行数 = %d, want 1", len(result.StallAggregates["鹏华"]))
	}
	if result.StallAggregates["鹏华"][0].Quantity != 5 {
		t.Errorf("数量 = %d, want 5", result.StallAggregates["鹏华"][0].Quantity)
	}
}

// 买家留言 / 卖家备注：同聚合键去重 + 多条合并；不同聚合键互不影响。
func TestProcessData_BuyerSellerNotes(t *testing.T) {
	engine := &Engine{
		Items: map[string]ConfigItem{
			"988808938880|喜马拉雅白鳄鱼": {Stall: "鹏华"},
		},
		Stalls: []string{"鹏华"},
	}
	headers := []string{"商品id", "商品规格", "商品数量", "买家留言", "卖家备注"}
	rows := [][]string{
		{"988808938880", "iPhone15|喜马拉雅白鳄鱼", "1", "要硬壳", "加急"},
		{"988808938880", "iPhone15|喜马拉雅白鳄鱼", "1", "要硬壳", "加急"},     // 完全重复：应被去重
		{"988808938880", "iPhone15|喜马拉雅白鳄鱼", "1", "要硅胶", "缺货换款"}, // 买家留言不同
		{"988808938880", "HuaweiPura70|喜马拉雅白鳄鱼", "1", "", "补发"},       // 不同型号：单独聚合
	}

	result := ProcessData(rows, headers, engine)
	penghua := result.StallAggregates["鹏华"]
	if len(penghua) != 2 {
		t.Fatalf("鹏华聚合行数 = %d, want 2", len(penghua))
	}

	// iPhone15 聚合行：买家留言 "要硬壳\n要硅胶"（顺序按首次出现），卖家备注 "加急"
	var iphone15 *AggregateRow
	for i := range penghua {
		if penghua[i].Model == "iPhone15" {
			iphone15 = &penghua[i]
			break
		}
	}
	if iphone15 == nil {
		t.Fatal("未找到 iPhone15 聚合行")
	}
	if iphone15.Quantity != 3 {
		t.Errorf("iPhone15 数量 = %d, want 3", iphone15.Quantity)
	}
	wantBuyer := "要硬壳\n要硅胶"
	if iphone15.BuyerNote != wantBuyer {
		t.Errorf("iPhone15 买家留言 = %q, want %q", iphone15.BuyerNote, wantBuyer)
	}
	wantSeller := "加急\n缺货换款"
	if iphone15.SellerNote != wantSeller {
		t.Errorf("iPhone15 卖家备注 = %q, want %q", iphone15.SellerNote, wantSeller)
	}

	// HuaweiPura70 聚合行：买家留言空，卖家备注 "补发"
	var huawei *AggregateRow
	for i := range penghua {
		if penghua[i].Model == "HuaweiPura70" {
			huawei = &penghua[i]
			break
		}
	}
	if huawei == nil {
		t.Fatal("未找到 HuaweiPura70 聚合行")
	}
	if huawei.BuyerNote != "" {
		t.Errorf("HuaweiPura70 买家留言 = %q, want 空", huawei.BuyerNote)
	}
	if huawei.SellerNote != "补发" {
		t.Errorf("HuaweiPura70 卖家备注 = %q, want %q", huawei.SellerNote, "补发")
	}
}

// 表头无买家留言/卖家备注列时，不报错，输出为空字符串。
func TestProcessData_NoNotesColumns(t *testing.T) {
	engine := &Engine{
		Items:  map[string]ConfigItem{"988808938880|单品": {Stall: "鹏华"}},
		Stalls: []string{"鹏华"},
	}
	headers := []string{"商品id", "商品规格", "商品数量"}
	rows := [][]string{
		{"988808938880", "单品", "2"},
	}
	result := ProcessData(rows, headers, engine)
	penghua := result.StallAggregates["鹏华"]
	if len(penghua) != 1 {
		t.Fatalf("鹏华聚合行数 = %d, want 1", len(penghua))
	}
	if penghua[0].BuyerNote != "" || penghua[0].SellerNote != "" {
		t.Errorf("留言/备注应为空, got buyer=%q seller=%q", penghua[0].BuyerNote, penghua[0].SellerNote)
	}
}

// 端到端：用 data/ 下的真实文件加载 + 处理
func TestLoadEngineAndProcess_RealFile(t *testing.T) {
	cfgPath := filepath.Join("..", "..", "data", "皮质壳配置表.xlsx")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Skip("配置文件不存在，跳过端到端测试")
	}

	engine, err := LoadEngine(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if len(engine.Stalls) < 2 {
		t.Errorf("档口数 = %d, want >= 2", len(engine.Stalls))
	}

	if len(engine.Items) == 0 {
		t.Fatal("配置项为空")
	}

	// 检查至少有一个有图
	withImg := 0
	for _, item := range engine.Items {
		if item.ImageBytes != nil {
			withImg++
		}
	}
	if withImg == 0 {
		t.Error("没有配置项包含图片数据")
	}
	t.Logf("加载 %d 个档口, %d 个配置项, %d 个有图", len(engine.Stalls), len(engine.Items), withImg)

	// 处理真实订单
	orderPath := filepath.Join("..", "..", "data", "皮质打图发货单测试.xlsx")
	if _, err := os.Stat(orderPath); os.IsNotExist(err) {
		t.Skip("订单文件不存在，跳过处理测试")
	}

	result, err := Process(orderPath, cfgPath)
	if err != nil {
		t.Fatalf("处理失败: %v", err)
	}

	if result.OutputPath == "" {
		t.Error("输出路径为空")
	}
	t.Logf("输出: %s, 档口数: %d, 未匹配: %d", result.OutputPath, len(result.StallAggregates), len(result.Unmatched))

	// 检查输出文件
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Errorf("输出文件不存在: %v", err)
	}
}