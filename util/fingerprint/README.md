# 设备指纹功能使用指南

## 一、概述

设备指纹功能用于为每个 WhatsApp 设备生成唯一的设备标识信息（设备型号、操作系统、语言、运营商等），避免所有设备使用相同指纹导致的风控问题。

### 核心特性

- ✅ **完全自动化**：只需配置地区，库内部自动处理所有逻辑
- ✅ **地区化支持**：根据地区生成符合该地区特征的设备指纹
- ✅ **零侵入**：不修改核心代码，不影响设备密钥生成
- ✅ **向后兼容**：未配置时行为不变

### 工作流程

```
创建 Container（配置地区）
  ↓
NewClient（自动检测配置）
  ↓
连接时自动读取/生成指纹
  ↓
应用到 ClientPayload
```

---

## 二、快速开始

### 2.1 基本使用（推荐）

```go
package main

import (
    "context"
    _ "go.mau.fi/whatsmeow/util/fingerprint/regions" // 必须导入以注册地区配置
    
    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/store/sqlstore"
    waLog "go.mau.fi/whatsmeow/util/log"
)

func main() {
    ctx := context.Background()
    logger := waLog.Stdout
    
    // 创建 Container 时配置地区（只需这一行！）
    container, err := sqlstore.New(
        ctx,
        "postgres",
        "postgres://user:pass@localhost/whatsmeow",
        logger,
        sqlstore.WithFingerprintRegion(sqlstore.Region_IN), // 印度地区
    )
    if err != nil {
        panic(err)
    }
    defer container.Close()
    
    // 获取设备
    device, err := container.GetFirstDevice(ctx)
    if err != nil {
        panic(err)
    }
    
    // 创建客户端（自动集成指纹，无需手动设置回调！）
    client := whatsmeow.NewClient(device, logger)
    
    // 连接（指纹会自动应用）
    err = client.Connect()
    if err != nil {
        panic(err)
    }
}
```

### 2.2 支持的地区

```go
sqlstore.Region_IN   // 印度（India）
sqlstore.Region_BR   // 巴西（Brazil）
sqlstore.Region_None // 不启用指纹（默认）
```

### 2.3 禁用指纹

```go
// 不传 WithFingerprintRegion 选项即可（默认不启用）
container, _ := sqlstore.New(ctx, dialect, address, log)
```

---

## 三、使用流程详解

### 3.1 单地区服务

**场景**：每个地区部署独立服务，服务只处理该地区的用户。

```go
// 创建 Container 时配置地区
container, _ := sqlstore.New(ctx, dialect, address, log,
    sqlstore.WithFingerprintRegion(sqlstore.Region_IN), // 印度
)

// 后续完全自动化
device, _ := container.GetFirstDevice(ctx)
client := whatsmeow.NewClient(device, log) // 自动集成指纹
client.Connect() // 指纹自动应用
```

### 3.2 多地区服务

**场景**：单个服务处理多个地区，根据用户信息选择地区。

```go
// 方式1：每个 Container 配置不同地区
containerIN, _ := sqlstore.New(ctx, dialect, address, log,
    sqlstore.WithFingerprintRegion(sqlstore.Region_IN),
)
containerBR, _ := sqlstore.New(ctx, dialect, address, log,
    sqlstore.WithFingerprintRegion(sqlstore.Region_BR),
)

// 方式2：环境变量配置
func createContainer(ctx context.Context, dialect, address string, log waLog.Logger) (*sqlstore.Container, error) {
    regionStr := os.Getenv("WHATSAPP_REGION")
    var region sqlstore.Region
    
    switch regionStr {
    case "IN":
        region = sqlstore.Region_IN
    case "BR":
        region = sqlstore.Region_BR
    default:
        region = sqlstore.Region_None
    }
    
    if region == sqlstore.Region_None {
        return sqlstore.New(ctx, dialect, address, log)
    }
    
    return sqlstore.New(ctx, dialect, address, log,
        sqlstore.WithFingerprintRegion(region),
    )
}
```

### 3.3 自动化流程说明

1. **Container 创建时**：配置存储在 `Container.FingerprintRegion`
2. **NewClient 时**：自动检测配置（`FingerprintRegion.IsValid()`）并设置 `GetClientPayload` 回调
3. **连接时**：
   - 从数据库读取已保存的指纹
   - 如果不存在，根据配置的地区自动生成
   - 应用指纹到 `ClientPayload`
   - 异步保存到数据库

**无需业务层任何额外代码！完全自动化！**

---

## 四、新增地区指南

### 4.1 步骤概览

新增地区只需 **3 步**：

1. 创建配置文件：`util/fingerprint/regions/xxx.go`
2. 实现配置：参考现有地区配置
3. 验证编译：`go build ./util/fingerprint/regions/...`

**无需修改其他代码！**

### 4.2 详细步骤

#### 步骤1：创建配置文件

在 `util/fingerprint/regions/` 目录下创建新文件，例如 `mexico.go`：

```go
package regions

import (
    "go.mau.fi/whatsmeow/util/fingerprint"
    "go.mau.fi/whatsmeow/proto/waCompanionReg"
    "go.mau.fi/whatsmeow/proto/waWa6"
)

func init() {
    config := &fingerprint.RegionConfig{
        Code: "MX", // ISO 3166-1 alpha-2 地区代码
        Name: "Mexico",
        
        // 1. 语言配置
        Languages: []fingerprint.LanguageConfig{
            {Code: "es", Weight: 1.0, Countries: []string{"MX"}}, // 西班牙语
        },
        
        // 2. 设备分布（按平台）
        DeviceDistribution: []fingerprint.PlatformDistribution{
            {
                Platform:     waWa6.ClientPayload_UserAgent_ANDROID,
                PlatformType: waCompanionReg.DeviceProps_ANDROID_PHONE,
                DeviceType:   waWa6.ClientPayload_UserAgent_PHONE,
                Weight:       0.75, // 75% Android
                OSName:       "Android",
                OSVersions:   []string{"11", "12", "13"},
            },
            {
                Platform:     waWa6.ClientPayload_UserAgent_IOS,
                PlatformType: waCompanionReg.DeviceProps_IOS_PHONE,
                DeviceType:   waWa6.ClientPayload_UserAgent_PHONE,
                Weight:       0.25, // 25% iOS
                OSName:       "iOS",
                OSVersions:   []string{"16", "17"},
            },
        },
        
        // 3. 移动网络配置（MCC/MNC）
        MobileNetworks: []fingerprint.MobileNetworkConfig{
            {MCC: "334", MNC: "020", OperatorName: "Telcel", Weight: 0.50},
            {MCC: "334", MNC: "030", OperatorName: "Movistar", Weight: 0.30},
            {MCC: "334", MNC: "050", OperatorName: "AT&T", Weight: 0.20},
        },
        
        // 4. 设备型号池（按制造商）
        DeviceModels: map[string][]fingerprint.DeviceModelConfig{
            "Samsung": {
                {Model: "Galaxy A32", ModelType: "SM-A325F", Board: "samsung"},
            },
            "Apple": {
                {Model: "iPhone 12", ModelType: "iPhone13,2", Board: "iPhone"},
            },
        },
    }
    
    _ = fingerprint.RegisterRegion(config)
}
```

#### 步骤2：在 Region 枚举中添加新地区

编辑 `store/sqlstore/region.go`：

```go
const (
    Region_None Region = iota
    Region_IN
    Region_BR
    Region_MX // 新增
)

func (r Region) String() string {
    switch r {
    case Region_IN:
        return "IN"
    case Region_BR:
        return "BR"
    case Region_MX: // 新增
        return "MX"
    default:
        return ""
    }
}
```

#### 步骤3：验证编译

```bash
go build ./util/fingerprint/regions/...
go build ./store/sqlstore/...
```

#### 步骤4：使用新地区

```go
container, _ := sqlstore.New(ctx, dialect, address, log,
    sqlstore.WithFingerprintRegion(sqlstore.Region_MX), // 使用新地区
)
```

### 4.3 配置数据来源

#### 设备分布数据
- **StatCounter**: https://gs.statcounter.com/os-market-share
- **GSMArena**: https://www.gsmarena.com/ 设备型号数据
- **地区市场报告**: 各地区的移动设备市场分析报告

#### 运营商数据（MCC/MNC）
- **ITU-T E.212 标准**: 官方移动国家码和网络码
- **MCC/MNC 数据库**: https://mcc-mnc.com/
- **运营商官网**: 各运营商官方信息

#### 语言数据
- **ISO 639-1**: 语言代码标准
- **CIA World Factbook**: 地区语言分布
- **地区语言统计数据**: 官方统计数据

### 4.4 配置验证规则

地区配置注册时会自动验证：

1. **语言权重总和** ≤ 1.0（允许小的浮点误差）
2. **平台分布权重总和** ≤ 1.0
3. **移动网络权重总和** ≤ 1.0
4. **地区代码** 不能为空

### 4.5 配置最佳实践

1. **设备分布**：
   - 参考真实市场数据
   - Android/iOS 比例要合理
   - 避免所有设备都是同一型号

2. **运营商配置**：
   - MCC 必须匹配国家代码
   - MNC 必须是该国家的真实运营商
   - 权重反映真实市场份额

3. **设备型号**：
   - 使用该地区常见的设备型号
   - 避免使用过新或过旧的型号
   - 保持型号多样性

4. **语言配置**：
   - 使用该地区主要语言
   - 权重反映语言使用比例
   - 国家代码必须匹配

---

## 五、数据库升级

### 5.1 自动升级

程序启动时会自动执行数据库升级：

```go
container, err := sqlstore.New(ctx, "postgres", "postgres://...", log)
// ↑ 这里会自动调用 container.Upgrade(ctx)，创建指纹表
```

### 5.2 验证升级

```sql
-- 检查表是否存在
SELECT EXISTS (
    SELECT FROM information_schema.tables 
    WHERE table_name = 'whatsmeow_device_fingerprint'
);

-- 查看版本
SELECT * FROM whatsmeow_version;
```

### 5.3 手动升级（可选）

如果不想等待程序自动升级，可以手动执行：

```bash
psql -h remote-host -U user -d whatsmeow -f store/sqlstore/upgrades/12-device-fingerprint.sql
```

---

## 六、常见问题

### Q1: 如何知道当前使用的地区？

```go
if container.FingerprintRegion.IsValid() {
    fmt.Println("Region:", container.FingerprintRegion.String())
} else {
    fmt.Println("Fingerprint disabled")
}
```

### Q2: 可以动态更改地区吗？

当前版本需要在创建 Container 时指定。如需动态更改，可以：
- 创建多个 Container（每个地区一个）
- 或直接修改 `container.FingerprintRegion`（不推荐，建议重新创建 Container）

### Q3: 如果业务层也设置了 GetClientPayload 会怎样？

Container 的自动配置会覆盖业务层设置。如果需要自定义逻辑，建议：
- 不启用 Container 的自动配置
- 在业务层的回调中调用指纹相关函数

### Q4: 性能影响？

- 指纹读取：每次连接时读取一次（已缓存）
- 指纹生成：仅在新设备时生成一次
- 影响可忽略

### Q5: 如何查看已注册的地区？

```go
import "go.mau.fi/whatsmeow/util/fingerprint"

regions := fingerprint.ListRegions()
fmt.Println(regions) // [IN BR]
```

### Q6: 地区代码不存在会怎样？

自动使用默认配置，不会报错。

### Q7: 如何验证生成的指纹是否正确？

```go
fp := fingerprint.GenerateFingerprint("IN")
fmt.Printf("Region: %s, Device: %s, Language: %s, MCC: %s\n",
    fp.LocaleCountry, fp.Device, fp.LocaleLanguage, fp.Mcc)
```

---

## 七、技术细节

### 7.1 设计原则

1. **零侵入核心代码**：不修改 `whatsmeow` 核心库代码
2. **独立存储**：新增独立数据库表，不修改现有表结构
3. **流程闭环**：生成 → 存储 → 读取 → 应用，无遗漏
4. **向后兼容**：现有设备不受影响，新设备自动启用
5. **容错降级**：任何环节失败都不影响正常连接

### 7.2 设备密钥生成流程（未修改）

**重要**：设备指纹功能**完全不会影响**设备密钥生成流程：

- ✅ `NewDevice()` 方法未被修改
- ✅ `NoiseKey`, `IdentityKey`, `RegistrationID`, `AdvSecretKey`, `SignedPreKey` 的生成逻辑保持不变
- ✅ 只在 `GetClientPayload` 回调中修改 `ClientPayload` 的**元数据字段**（设备信息），不涉及密钥

### 7.3 容错机制

| 场景 | 处理方式 | 影响 |
|------|----------|------|
| 数据库连接失败 | 使用默认 payload | 无影响，可正常连接 |
| 指纹读取失败 | 使用默认 payload | 无影响，下次重新生成 |
| 指纹生成失败 | 使用默认 payload | 无影响，下次重试 |
| 指纹保存失败 | 记录日志，继续 | 无影响，下次重新生成 |
| 数据库表不存在 | 使用默认 payload | 无影响，升级后自动创建 |

---

## 八、文件结构

```
util/fingerprint/
├── README.md           # 本文档
├── generator.go        # 通用生成器（配置驱动）
├── region.go          # 地区配置结构定义
├── registry.go        # 地区注册表
├── apply.go           # 指纹应用逻辑
└── regions/           # 地区配置目录
    ├── default.go     # 默认配置
    ├── india.go       # 印度配置
    └── brazil.go      # 巴西配置

store/sqlstore/
├── region.go          # 地区枚举定义
├── fingerprint.go     # 指纹存储实现
└── container.go       # Container（包含 FingerprintRegion 字段）

store/sqlstore/upgrades/
└── 12-device-fingerprint.sql  # 数据库迁移文件
```

---

## 九、相关文档

- **数据库升级机制**: 参考 `DATABASE_UPGRADE_GUIDE.md`
- **业务层使用指南**: 参考 `BUSINESS_USAGE_GUIDE.md`
- **技术方案详情**: 参考 `FINAL_TECHNICAL_PLAN.md`

---

**文档版本**: v1.0  
**最后更新**: 2024-12-XX  
**支持地区**: IN（印度）、BR（巴西）  
**状态**: ✅ 已实施，可直接使用
