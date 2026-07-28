# Banner 管理接口

本文档描述管理后台 Banner 的查询、创建、编辑、删除及投放选项接口。管理接口统一使用 `/admin` 前缀，需要管理员登录；受权限控制的操作还需具备对应的 Banner 权限。

## 接口清单

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/banners` | 分页查询 Banner |
| `GET` | `/admin/banners/delivery-options` | 查询“应用 → 应用包 → 包版本”级联选项 |
| `GET` | `/admin/banners/:id` | 查询 Banner 详情 |
| `POST` | `/admin/banners` | 创建 Banner |
| `PUT` | `/admin/banners/:id` | 更新 Banner，并整体替换投放范围 |
| `DELETE` | `/admin/banners/:id` | 删除 Banner 及其投放关联 |

## 分页查询

`GET /admin/banners`

| Query 参数 | 必填 | 说明 |
| --- | ---: | --- |
| `page` | 否 | 页码，默认 `1` |
| `page_size` | 否 | 每页数量，默认 `20` |
| `position_key` | 否 | 展示位置标识 |
| `country_code` | 否 | ISO 3166-1 alpha-2 国家或地区代码 |
| `app_code` | 否 | 应用代码 |
| `package_code` | 否 | 应用包代码 |
| `version_code` | 否 | 包版本号；绑定“全部版本”的 Banner 也可命中 |
| `jump_type` | 否 | 跳转类型：`1` 链接、`2` 模板、`3` 文生图、`4` 文生视频 |
| `status` | 否 | `0` 禁用、`1` 启用 |
| `keyword` | 否 | 按 Banner 名称或备注模糊查询 |

## 投放级联选项

`GET /admin/banners/delivery-options`

仅返回启用的应用、应用包和包版本。响应 `data` 的结构为：

```json
[
  {
    "app_code": "ai-video",
    "app_name": "AI Video",
    "packages": [
      {
        "package_code": "com.example.video",
        "package_name": "Android 正式包",
        "versions": [
          { "version_code": "1.2.0" },
          { "version_code": "1.3.0" }
        ]
      }
    ]
  }
]
```

## 创建与更新

- 创建：`POST /admin/banners`
- 更新：`PUT /admin/banners/:id`

两者使用相同的 JSON 请求结构。更新接口会整体替换已有投放范围。

```json
{
  "name": "首页夏日活动",
  "cover_image": "https://cdn.example.com/banners/summer.jpg",
  "display_position_keys": ["home_banner"],
  "country_codes": ["CN", "US"],
  "app_targets": [
    {
      "app_code": "ai-video",
      "package_code": "com.example.video",
      "version_codes": ["1.2.0", "1.3.0"]
    }
  ],
  "remark": "首页活动投放",
  "sort": 10,
  "jump_type": 2,
  "jump_url": "",
  "template_id": 42,
  "status": 1,
  "subscription_status": 3
}
```

### 投放范围字段

| 字段 | 说明 |
| --- | --- |
| `display_position_keys` | 展示位置标识数组；空数组表示全部展示位置，不写入展示位置关联数据 |
| `country_codes` | 国家代码数组；空数组表示全部国家，不写入国家关联数据 |
| `app_targets` | 应用包投放数组；空数组表示全部应用、包和版本，不写入应用/包/版本关联数据 |
| `app_targets[].app_code` | 必填，必须是启用的应用 |
| `app_targets[].package_code` | 必填，必须属于所选应用且处于启用状态 |
| `app_targets[].version_codes` | 可空；空数组表示所选包的全部版本，不写入具体版本关联数据 |

同一个应用包重复出现时，服务端会合并版本并去重。如果同一包同时提交空版本数组和具体版本，空数组优先，最终表示该包全部版本。

### 会员类型

`subscription_status` 必填：

- `1`：非会员
- `2`：会员
- `3`：全部用户

### 跳转字段

| `jump_type` | 必填字段 | 服务端处理 |
| ---: | --- | --- |
| `1` | `jump_url` | 接受绝对 URL、深链或以 `/` 开头的站内路径；清空 `template_id` |
| `2` | `template_id` | 目标模板必须存在；清空 `jump_url` |
| `3` | 无 | 跳转文生图；清空 `jump_url` 和 `template_id` |
| `4` | 无 | 跳转文生视频；清空 `jump_url` 和 `template_id` |

## 客户端查询

客户端展示接口为 `GET /api/banners/list`，详细的公共 Header、投放匹配规则和响应结构见 [客户端 Banner API](client-banners-api.md)。

