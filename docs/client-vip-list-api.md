# VIP 套餐列表接口

## 接口

`GET /api/vip/list`

需要 `Authorization: Bearer <JWT>` 及客户端公共 Header。

## 查询参数

| 参数 | 必填 | 说明 |
| --- | ---: | --- |
| `vip_types` | 是 | 套餐类型数组，至少一项；多值示例：`vip_types=1&vip_types=2` |

接口结合 `Video_App_Code`、`Video_App_Package_Code`、`Video_App_Version` 和当前登录用户状态筛选套餐，仅返回 `status=1`、`display_mode=1` 的记录，排序为 `is_default DESC, sort DESC, id DESC`。未订阅用户返回首订价格和首订赠送积分，其他用户返回续订价格和订阅赠送积分。

## 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 2,
      "vip_type": 2,
      "suk_code": "222222",
      "name": "首页OB拦截套餐",
      "level_name": "普通套餐",
      "currency": "USD",
      "vip_duration_days": 1,
      "trial_days": 0,
      "badge_text": "",
      "agreement_default_checked": 0,
      "display_mode": 1,
      "status": 1,
      "free_trial": 0,
      "is_subscription": 1,
      "is_default": 0,
      "subscription_description": "",
      "subscription_price": 0,
      "original_price": 0,
      "subscription_points": 0,
      "subscription_period": 1,
      "sort": 0,
      "description": "",
      "remark": "",
      "created_at": 1784859371,
      "updated_at": 1784835434
    }
  ]
}
```

时间字段为 Unix 秒。
