// node:test 单元测试：peijian 配置解析纯函数
// 运行：node --test web/
const { test } = require('node:test');
const assert = require('node:assert');
const { cleanAccessoryName, parsePeijianConfigSheet } = require('./app.js');

test('cleanAccessoryName: 去「单独」前缀 + 「不含壳」尾注', () => {
  const cases = [
    ['单独青苹果支架 -不含壳', '青苹果支架'],
    ['单独青苹果支架-不含壳', '青苹果支架'],
    ['青苹果支架 -不含壳', '青苹果支架'],
    ['布丁夹心鲷鱼烧支架 不含壳', '布丁夹心鲷鱼烧支架'], // 尾注不带 -
    ['单独布丁夹心鲷鱼烧支架不含壳', '布丁夹心鲷鱼烧支架'],
    ['单独青苹果支架', '青苹果支架'],
    ['青苹果支架', '青苹果支架'],
    [' 单独 青苹果支架 -不含壳 ', '青苹果支架'],
    ['N52强力磁吸软胶吸盘-不含壳 ', 'N52强力磁吸软胶吸盘'], // 尾部不换行空格
    ['', ''],
    [null, ''],
  ];
  for (const [input, want] of cases) {
    assert.strictEqual(cleanAccessoryName(input), want, `cleanAccessoryName(${JSON.stringify(input)})`);
  }
});

test('parsePeijianConfigSheet: 别名 sheet 解析 + 不污染档口', async () => {
  const sheets = {
    '支架编码': [
      ['商品ID', 'SKU名称', '编码1', '编码2'],
      ['123', '壳+配件1+配件2', 'CODE1', 'CODE2'],
      ['456', '单品配件', 'CODE3', ''],
    ],
    '支架档口': [
      ['档口A', '档口B'],
      ['CODE1', 'CODE2'],
      ['CODE3', ''],
    ],
    '配件别名': [
      ['奶黄推拉支架', '奶黄糯米糍支架'],
      ['奶黄色推拉支架', '芒果糯米糍支架'],
      ['奶油黄推拉支架', null],
    ],
  };
  const sheetNames = ['支架编码', '支架档口', '配件别名'];

  const cfg = await parsePeijianConfigSheet(sheets, sheetNames);

  // mapping：key 小写化，值为编码数组
  assert.deepStrictEqual(cfg.mapping['123|壳+配件1+配件2'], ['CODE1', 'CODE2']);
  assert.deepStrictEqual(cfg.mapping['456|单品配件'], ['CODE3']);

  // stalls：编码小写 → 档口名；「配件别名」不得混入
  assert.strictEqual(cfg.stalls['code1'], '档口A');
  assert.strictEqual(cfg.stalls['code2'], '档口B');
  assert.strictEqual(cfg.stalls['code3'], '档口A');

  // stallOrder：仅真实档口，不含别名 sheet 的列
  assert.deepStrictEqual(cfg.stallOrder, ['档口A', '档口B']);
  assert.ok(!cfg.stallOrder.includes('奶黄推拉支架'), '别名不应出现在 stallOrder');

  // aliases：同列下方各行 → 列首标准名（标准名映射到自身）
  assert.deepStrictEqual(cfg.aliases, {
    '奶黄推拉支架': '奶黄推拉支架',
    '奶黄色推拉支架': '奶黄推拉支架',
    '奶油黄推拉支架': '奶黄推拉支架',
    '奶黄糯米糍支架': '奶黄糯米糍支架',
    '芒果糯米糍支架': '奶黄糯米糍支架',
  });
});

test('parsePeijianConfigSheet: 无「配件别名」sheet 时 aliases 为空', async () => {
  const sheets = {
    '支架编码': [
      ['商品ID', 'SKU名称', '编码1'],
      ['123', '壳+配件1', 'CODE1'],
    ],
    '支架档口': [
      ['档口A'],
      ['CODE1'],
    ],
  };
  const cfg = await parsePeijianConfigSheet(sheets, ['支架编码', '支架档口']);
  assert.deepStrictEqual(cfg.aliases, {});
  assert.deepStrictEqual(cfg.stallOrder, ['档口A']);
});
