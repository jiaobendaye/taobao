// Package datu 打图工厂分配测试
package datu

import (
	"os"
	"path/filepath"
	"testing"
)

// ---- ParseDatuCode ----

func TestParseDatuCode(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantMaterial    string
		wantCode        string
	}{
		{"标准格式带连字符", "【DYT彩银白色-DTY7958】", "DYT彩银白色", "DTY7958"},
		{"无连字符", "【DTY彩银白色DTY7958】", "", ""},
		{"缺少右括号", "【DYT彩银白色-DTY7958", "", ""},
		{"空字符串", "", "", ""},
		{"多个连字符 SplitN 限 2", "【AB-CD-EF】", "AB", "CD-EF"},
		{"带前后空格", "  【PH仓-皮质硬壳】  ", "PH仓", "皮质硬壳"},
		{"只有方括号无内容", "【】", "", ""},
		{"方括号中间单字段", "【PH皮质】", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, c := ParseDatuCode(tt.input)
			if m != tt.wantMaterial {
				t.Errorf("material = %q, want %q", m, tt.wantMaterial)
			}
			if c != tt.wantCode {
				t.Errorf("code = %q, want %q", c, tt.wantCode)
			}
		})
	}
}

// ---- ProcessData 聚合 ----

func newEngine() *Engine {
	return &Engine{
		FactoryByProductID: map[string]string{
			"1050879735957": "打图工厂1",
			"1053838905482": "打图工厂1",
			"1058553670530": "打图工厂1",
			"1058691249562": "打图工厂1",
		},
		Factories: []string{"打图工厂1"},
	}
}

func TestProcessData_Aggregate(t *testing.T) {
	engine := newEngine()
	headers := []string{"商品id", "商品规格", "商品规格商家编码", "商品数量"}
	rows := [][]string{
		{"1050879735957", "小米 14|薄荷海", "【DYT彩银白色-DTY7958】", "1"},
		{"1050879735957", "小米 14|薄荷海", "【DYT彩银白色-DTY7958】", "2"},
		{"1053838905482", "华为 Mate 60 Pro|薄荷海", "【DYT彩银白色-DTY7958】", "3"},
	}
	result := ProcessData(rows, headers, engine)
	got := result.FactoryAggregates["打图工厂1"]
	if len(got) != 2 {
		t.Fatalf("rows = %d, want 2", len(got))
	}
	// 找 小米14 那行
	var xiaomiRow *AggregateRow
	for i := range got {
		if got[i].Model == "小米14" {
			xiaomiRow = &got[i]
		}
	}
	if xiaomiRow == nil {
		t.Fatal("未找到 小米14 行")
	}
	if xiaomiRow.Quantity != 3 {
		t.Errorf("数量 = %d, want 3", xiaomiRow.Quantity)
	}
	if xiaomiRow.Code != "DTY7958" || xiaomiRow.Material != "DYT彩银白色" {
		t.Errorf("编码/素材 = (%q,%q), want (DTY7958,DYT彩银白色)", xiaomiRow.Code, xiaomiRow.Material)
	}
	if xiaomiRow.Name != DefaultName {
		t.Errorf("姓名 = %q, want %q", xiaomiRow.Name, DefaultName)
	}
}

func TestProcessData_DifferentCodeSplits(t *testing.T) {
	engine := newEngine()
	headers := []string{"商品id", "商品规格", "商品规格商家编码", "商品数量"}
	rows := [][]string{
		{"1050879735957", "小米 14|薄荷海", "【DYT彩银白色-DTY7958】", "1"},
		{"1050879735957", "小米 14|薄荷海", "【DYT冰雾魔方透白-DTY7959】", "1"},
	}
	result := ProcessData(rows, headers, engine)
	got := result.FactoryAggregates["打图工厂1"]
	if len(got) != 2 {
		t.Fatalf("rows = %d, want 2 (不同编码应拆行)", len(got))
	}
}

func TestProcessData_DifferentMaterialSplits(t *testing.T) {
	engine := newEngine()
	headers := []string{"商品id", "商品规格", "商品规格商家编码", "商品数量"}
	rows := [][]string{
		{"1050879735957", "小米 14|薄荷海", "【DYT彩银白色-DTY7958】", "1"},
		{"1050879735957", "小米 14|薄荷海", "【DTY彩银白色-DTY7958】", "1"},
	}
	result := ProcessData(rows, headers, engine)
	got := result.FactoryAggregates["打图工厂1"]
	if len(got) != 2 {
		t.Fatalf("rows = %d, want 2 (不同素材应拆行)", len(got))
	}
}

func TestProcessData_UnmatchedCodeProducesEmptyFields(t *testing.T) {
	engine := newEngine()
	headers := []string{"商品id", "商品规格", "商品规格商家编码", "商品数量"}
	rows := [][]string{
		{"1050879735957", "小米 14|薄荷海", "【PH皮质】", "1"}, // 无连字符
	}
	result := ProcessData(rows, headers, engine)
	got := result.FactoryAggregates["打图工厂1"]
	if len(got) != 1 {
		t.Fatalf("rows = %d, want 1", len(got))
	}
	if got[0].Code != "" || got[0].Material != "" {
		t.Errorf("编码/素材应为空, got (%q,%q)", got[0].Code, got[0].Material)
	}
	if got[0].Model != "小米14" {
		t.Errorf("手机型号 = %q, want 小米14", got[0].Model)
	}
}

func TestProcessData_SkipNonDatuOrders(t *testing.T) {
	engine := newEngine()
	headers := []string{"商品id", "商品规格", "商品规格商家编码", "商品数量"}
	rows := [][]string{
		{"999999999999", "iPhone 15|某SKU", "【PH仓-皮质硬壳】", "1"}, // 不在 Engine
		{"1050879735957", "小米 14|薄荷海", "【DYT彩银白色-DTY7958】", "1"},
	}
	result := ProcessData(rows, headers, engine)
	got := result.FactoryAggregates["打图工厂1"]
	if len(got) != 1 {
		t.Fatalf("rows = %d, want 1 (非打图订单应跳过)", len(got))
	}
}

func TestProcessData_MultipleFactories(t *testing.T) {
	engine := &Engine{
		FactoryByProductID: map[string]string{
			"1111111111111": "打图工厂1",
			"2222222222222": "打图工厂2",
		},
		Factories: []string{"打图工厂1", "打图工厂2"},
	}
	headers := []string{"商品id", "商品规格", "商品规格商家编码", "商品数量"}
	rows := [][]string{
		{"1111111111111", "iPhone15|A", "【X-001】", "2"},
		{"2222222222222", "华为Pura70|B", "【Y-002】", "3"},
	}
	result := ProcessData(rows, headers, engine)
	if len(result.FactoryAggregates["打图工厂1"]) != 1 {
		t.Errorf("打图工厂1 rows = %d, want 1", len(result.FactoryAggregates["打图工厂1"]))
	}
	if len(result.FactoryAggregates["打图工厂2"]) != 1 {
		t.Errorf("打图工厂2 rows = %d, want 1", len(result.FactoryAggregates["打图工厂2"]))
	}
	if result.FactoryAggregates["打图工厂1"][0].Quantity != 2 {
		t.Errorf("打图工厂1 数量 = %d, want 2", result.FactoryAggregates["打图工厂1"][0].Quantity)
	}
}

func TestProcessData_QuantityFallback(t *testing.T) {
	engine := newEngine()
	headers := []string{"商品id", "商品规格", "商品规格商家编码"} // 无商品数量列
	rows := [][]string{
		{"1050879735957", "小米 14|薄荷海", "【X-001】"},
	}
	result := ProcessData(rows, headers, engine)
	got := result.FactoryAggregates["打图工厂1"]
	if len(got) != 1 {
		t.Fatalf("rows = %d, want 1", len(got))
	}
	if got[0].Quantity != 1 {
		t.Errorf("数量缺省应为 1, got %d", got[0].Quantity)
	}
}

func TestLookupFactory(t *testing.T) {
	engine := newEngine()
	if got := engine.LookupFactory("1050879735957"); got != "打图工厂1" {
		t.Errorf("已知 ID = %q, want 打图工厂1", got)
	}
	// 大小写不敏感
	if got := engine.LookupFactory("1050879735957"); got != "打图工厂1" {
		t.Errorf("大小写不敏感失败: %q", got)
	}
	if got := engine.LookupFactory("999999999999"); got != "" {
		t.Errorf("未知 ID 应返回空, got %q", got)
	}
}

// ---- 端到端：data/ 下的真实文件 ----

func TestLoadEngineAndProcess_RealFile(t *testing.T) {
	cfgPath := filepath.Join("..", "..", "data", "打图工厂配置表.xlsx")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Skip("打图工厂配置文件不存在，跳过端到端测试")
	}

	engine, err := LoadEngine(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if len(engine.Factories) == 0 {
		t.Fatal("未识别到任何工厂")
	}
	if len(engine.FactoryByProductID) == 0 {
		t.Fatal("未识别到任何商品ID")
	}
	t.Logf("加载 %d 个工厂, %d 个商品ID", len(engine.Factories), len(engine.FactoryByProductID))

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
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Errorf("输出文件不存在: %v", err)
	}
	t.Logf("输出: %s, 工厂数: %d, 聚合行数: %d",
		result.OutputPath, len(result.FactoryAggregates), totalAggRows(result))
}

func totalAggRows(r *Result) int {
	n := 0
	for _, rows := range r.FactoryAggregates {
		n += len(rows)
	}
	return n
}
