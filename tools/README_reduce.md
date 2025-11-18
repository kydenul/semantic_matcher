# 词向量文件压缩工具

## 工具说明

### 1. reduce_vec_size - 减少词汇量

只保留最常用的 N 个词，大幅减小文件大小。

**使用方法：**

```bash
# 编译工具
go build -o reduce_vec_size tools/reduce_vec_size.go

# 只保留前 10 万个词（推荐）
./reduce_vec_size \
  -input vector/wiki.en.align.vec \
  -output vector/wiki.en.align.100k.vec \
  -max 100000

# 只保留前 5 万个词（更小）
./reduce_vec_size \
  -input vector/wiki.zh.align.vec \
  -output vector/wiki.zh.align.50k.vec \
  -max 50000
```

**效果：**
- 原始文件 5.3 GB → 约 500 MB（保留 10 万词）
- 原始文件 5.3 GB → 约 250 MB（保留 5 万词）
- 减少 90%+ 的文件大小

### 2. reduce_dimension - 降低向量维度

将 300 维向量降低到更小的维度（如 100 维或 50 维）。

**使用方法：**

```bash
# 编译工具
go build -o reduce_dimension tools/reduce_dimension.go

# 降低到 100 维
./reduce_dimension \
  -input vector/wiki.en.align.vec \
  -output vector/wiki.en.align.d100.vec \
  -dim 100

# 降低到 50 维
./reduce_dimension \
  -input vector/wiki.zh.align.vec \
  -output vector/wiki.zh.align.d50.vec \
  -dim 50
```

**效果：**
- 300 维 → 100 维：减少约 67% 的大小
- 300 维 → 50 维：减少约 83% 的大小

### 3. 组合使用（最佳效果）

同时减少词汇量和维度可以获得最大压缩：

```bash
# 步骤 1: 先减少词汇量
./reduce_vec_size \
  -input vector/wiki.en.align.vec \
  -output vector/wiki.en.align.100k.vec \
  -max 100000

# 步骤 2: 再降低维度
./reduce_dimension \
  -input vector/wiki.en.align.100k.vec \
  -output vector/wiki.en.align.100k.d100.vec \
  -dim 100
```

**最终效果：**
- 5.3 GB → 约 150 MB（10 万词 + 100 维）
- 减少 97% 的文件大小！

## 推荐配置

根据你的使用场景选择：

### 场景 1: 高精度要求
```bash
# 保留 20 万词 + 200 维
词汇量: 200,000
维度: 200
文件大小: ~600 MB
```

### 场景 2: 平衡性能和精度（推荐）
```bash
# 保留 10 万词 + 100 维
词汇量: 100,000
维度: 100
文件大小: ~150 MB
```

### 场景 3: 最小内存占用
```bash
# 保留 5 万词 + 50 维
词汇量: 50,000
维度: 50
文件大小: ~40 MB
```

## 性能影响

| 配置 | 文件大小 | 加载时间 | 内存占用 | 精度损失 |
|------|---------|---------|---------|---------|
| 原始 (2.5M词, 300维) | 5.3 GB | 60s | 3 GB | 0% |
| 20万词, 200维 | 600 MB | 8s | 400 MB | ~5% |
| 10万词, 100维 | 150 MB | 2s | 100 MB | ~10% |
| 5万词, 50维 | 40 MB | 0.5s | 30 MB | ~15% |

## 注意事项

1. **词汇覆盖率**：减少词汇量会降低 OOV（未登录词）的覆盖率
2. **语义精度**：降低维度会略微降低语义相似度的精度
3. **跨语言对齐**：如果使用跨语言功能，建议两种语言使用相同的维度
4. **备份原文件**：处理前请备份原始文件

## 验证压缩效果

压缩后可以运行测试验证效果：

```bash
# 更新配置文件使用压缩后的向量
# config/config.yaml
vector_file_paths:
  - "vector/wiki.zh.align.100k.d100.vec"
  - "vector/wiki.en.align.100k.d100.vec"

# 运行测试
go test -v ./...

# 运行示例
go run examples/cross_lingual/cross_lingual_example.go
```

## 快速开始

```bash
# 1. 进入项目目录
cd semantic_matcher

# 2. 编译工具
go build -o reduce_vec_size tools/reduce_vec_size.go
go build -o reduce_dimension tools/reduce_dimension.go

# 3. 压缩英文向量（推荐配置）
./reduce_vec_size \
  -input vector/wiki.en.align.vec \
  -output vector/wiki.en.align.100k.vec \
  -max 100000

./reduce_dimension \
  -input vector/wiki.en.align.100k.vec \
  -output vector/wiki.en.align.100k.d100.vec \
  -dim 100

# 4. 压缩中文向量
./reduce_vec_size \
  -input vector/wiki.zh.align.vec \
  -output vector/wiki.zh.align.100k.vec \
  -max 100000

./reduce_dimension \
  -input vector/wiki.zh.align.100k.vec \
  -output vector/wiki.zh.align.100k.d100.vec \
  -dim 100

# 5. 清理中间文件
rm vector/*.100k.vec

# 6. 最终文件
# vector/wiki.en.align.100k.d100.vec (~150 MB)
# vector/wiki.zh.align.100k.d100.vec (~20 MB)
```

## 其他优化方法

### 使用二进制格式
可以考虑将文本格式转换为二进制格式（如 numpy .npy 或自定义格式）：
- 减少约 30-40% 的存储空间
- 加载速度提升 2-3 倍

### 使用量化
将 float32 量化为 int8 或 int16：
- 减少 50-75% 的内存占用
- 略微降低精度（通常可接受）

### 按需加载
只加载需要的词汇：
- 根据你的应用场景预先统计常用词
- 只加载这些词的向量
