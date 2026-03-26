"""
生成模拟酒店入住率数据
输出：hotel_data.csv, summary.json（项目根目录）
"""

import json

import numpy as np
import pandas as pd

np.random.seed(42)

HOTELS = 10
BRANDS = ["economy", "midscale", "premium"]
DISTRICTS = ["cbd", "suburban", "transport_hub"]

hn_prefixes = ["金", "银", "悦", "瑞", "盛", "华", "锦", "君", "雅", "泰"]
hn_types = ["都", "城", "苑", "庭", "阁", "府", "居", "轩"]
hn_suffixes = ["酒店", "宾馆", "大酒店", "商务酒店", "度假酒店"]

names_set = set()
while len(names_set) < HOTELS:
    names_set.add(
        np.random.choice(hn_prefixes) +
        np.random.choice(hn_types) +
        np.random.choice(hn_suffixes)
    )
hotel_names = sorted(list(names_set))

hotel_meta = {
    f"H{h:03d}": {
        "base": np.random.uniform(0.45, 0.85),
        "brand": np.random.choice(BRANDS),
        "district": np.random.choice(DISTRICTS),
        "name": name,
    }
    for h, name in enumerate(hotel_names, start=1)
}

dates = pd.date_range("2025-01-01", "2025-12-31", freq="D")

rows = []
for date in dates:
    holiday_boost = 0.0
    if (date.month == 1 and date.day >= 28) or (date.month == 2 and date.day <= 4):
        holiday_boost = 0.15
    elif date.month == 5 and date.day <= 5:
        holiday_boost = 0.10
    elif date.month == 10 and date.day <= 8:
        holiday_boost = 0.12

    weekend_boost = 0.05 if date.weekday() >= 5 else 0.0
    seasonal = 0.08 * np.sin(2 * np.pi * (date.dayofyear / 365))

    for hotel_id, meta in hotel_meta.items():
        noise = np.random.normal(0.0, 0.04)
        occ = meta["base"] + seasonal + holiday_boost + weekend_boost + noise
        occ = round(float(np.clip(occ, 0.0, 1.0)), 4)
        rows.append({
            "hotel_id": hotel_id,
            "hotel_name": meta["name"],
            "date": date.strftime("%Y-%m-%d"),
            "occupancy": occ,
            "brand_tier": meta["brand"],
            "district": meta["district"],
        })

df = pd.DataFrame(rows)
df.to_csv("hotel_data.csv", index=False)
print(f"生成完毕：{len(df)} 行，{HOTELS} 家酒店，{len(dates)} 天")


# ========== 生成 summary.json ==========

def desc(series):
    return {
        "mean": round(series.mean(), 4),
        "std": round(series.std(), 4),
        "min": round(series.min(), 4),
        "max": round(series.max(), 4),
        "count": int(series.count()),
    }


summary = {}

# 1. 全局
summary["overall"] = desc(df["occupancy"])

# 2. 按品牌
summary["by_brand"] = {
    brand: desc(g["occupancy"])
    for brand, g in df.groupby("brand_tier")
}

# 3. 按区域
summary["by_district"] = {
    district: desc(g["occupancy"])
    for district, g in df.groupby("district")
}

# 4. 按月（全局）
df["month"] = df["date"].str[:7]
summary["by_month"] = {
    month: desc(g["occupancy"])
    for month, g in df.groupby("month")
}

# 5. 按酒店（全年 + 按月）
summary["by_hotel"] = {}
for hotel_id, hg in df.groupby("hotel_id"):
    hotel_name = hg["hotel_name"].iloc[0]
    summary["by_hotel"][hotel_id] = {
        "name": hotel_name,
        "brand": hg["brand_tier"].iloc[0],
        "district": hg["district"].iloc[0],
        "annual": desc(hg["occupancy"]),
        "by_month": {
            month: desc(mg["occupancy"])
            for month, mg in hg.groupby("month")
        },
    }

with open("summary.json", "w", encoding="utf-8") as f:
    json.dump(summary, f, ensure_ascii=False, indent=2)

print("summary.json 生成完毕")
print(f"\n入住率分布：\n{df['occupancy'].describe().round(3)}")
