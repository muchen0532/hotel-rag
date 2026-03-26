"""
gen_embeddings.py
把 hotel_data.csv 向量化后存入 Qdrant（使用 Jina Embedding API）

依赖安装：
    pip install qdrant-client requests pandas tqdm
"""

import argparse
import json
import os
import time

import pandas as pd
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry
from qdrant_client import QdrantClient
from qdrant_client.models import Distance, VectorParams, PointStruct
from tqdm import tqdm

EMBEDDING_MODEL = "jina-embeddings-v3"
EMBEDDING_URL = "https://api.jina.ai/v1/embeddings"
VECTOR_DIM = 1024
BATCH_SIZE = 5
CHECKPOINT_FILE = "checkpoint.json"
PROXIES = {
    "http": "http://127.0.0.1:7897",
    "https": "http://127.0.0.1:7897",
}

DISTRICT_ZH = {
    "cbd": "CBD商业区",
    "suburban": "郊区",
    "transport_hub": "交通枢纽",
}

BRAND_ZH = {
    "economy": "经济型",
    "midscale": "中档",
    "premium": "高档",
}


def row_to_text(row: pd.Series) -> str:
    district = DISTRICT_ZH.get(row["district"], row["district"])
    brand = BRAND_ZH.get(row["brand_tier"], row["brand_tier"])
    occupancy_pct = f"{row['occupancy'] * 100:.1f}%"
    return (
        f"{row['hotel_name']}（{row['hotel_id']}）位于{district}，"
        f"品牌档次{brand}，"
        f"{row['date']}的入住率为{occupancy_pct}。"
    )


def make_session() -> requests.Session:
    session = requests.Session()
    retry = Retry(total=3, backoff_factor=2, status_forcelist=[500, 502, 503, 504])
    session.mount("https://", HTTPAdapter(max_retries=retry))
    return session


def embed_batch(session: requests.Session, api_key: str, texts: list[str]) -> list[list[float]]:
    resp = session.post(
        EMBEDDING_URL,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        },
        json={
            "model": EMBEDDING_MODEL,
            "input": texts,
            "task": "retrieval.passage",
        },
        proxies=PROXIES,
        timeout=60,
    )
    resp.raise_for_status()
    data = resp.json()
    return [item["embedding"] for item in data["data"]]


def save_checkpoint(batch_index: int):
    with open(CHECKPOINT_FILE, "w") as f:
        json.dump({"batch": batch_index}, f)


def load_checkpoint() -> int:
    if os.path.exists(CHECKPOINT_FILE):
        with open(CHECKPOINT_FILE) as f:
            return json.load(f)["batch"]
    return 0


def main(csv_path: str, collection: str, qdrant_url: str):
    api_key = os.environ.get("JINA_API_KEY")
    if not api_key:
        raise ValueError("请先设置环境变量 JINA_API_KEY")

    df = pd.read_csv(csv_path)
    print(f"读取 {len(df)} 条记录，字段：{list(df.columns)}")

    qdrant = QdrantClient(url=qdrant_url, timeout=120)
    collection = f"{collection}_{VECTOR_DIM}"

    if qdrant.collection_exists(collection):
        confirm = input(f"Collection '{collection}' 已存在，确认删除重建？(y/n): ")
        if confirm != "y":
            return
        qdrant.delete_collection(collection)
        # 删除重建时清掉旧断点
        if os.path.exists(CHECKPOINT_FILE):
            os.remove(CHECKPOINT_FILE)

    qdrant.create_collection(
        collection_name=collection,
        vectors_config=VectorParams(size=VECTOR_DIM, distance=Distance.COSINE),
    )
    print(f"Collection '{collection}' 创建完成")

    texts = [row_to_text(row) for _, row in df.iterrows()]
    session = make_session()
    start_batch = load_checkpoint()
    batches = list(range(0, len(texts), BATCH_SIZE))

    if start_batch > 0:
        print(f"从断点继续，跳过前 {start_batch} 批")

    for batch_index in tqdm(batches, desc="Embedding"):
        if batch_index < start_batch:
            continue

        batch_texts = texts[batch_index: batch_index + BATCH_SIZE]
        batch_rows = df.iloc[batch_index: batch_index + BATCH_SIZE]

        vectors = embed_batch(session, api_key, batch_texts)

        points = [
            PointStruct(
                id=batch_index + i,
                vector=vec,
                payload={
                    "hotel_id": row["hotel_id"],
                    "hotel_name": row["hotel_name"],
                    "date": row["date"],
                    "occupancy": float(row["occupancy"]),
                    "brand_tier": row["brand_tier"],
                    "district": row["district"],
                    "text": batch_texts[i],
                },
            )
            for i, (vec, (_, row)) in enumerate(zip(vectors, batch_rows.iterrows()))
        ]

        qdrant.upsert(collection_name=collection, points=points)
        save_checkpoint(batch_index + BATCH_SIZE)
        time.sleep(0.2)  # 避免请求过快

    # 完成后清掉断点文件
    if os.path.exists(CHECKPOINT_FILE):
        os.remove(CHECKPOINT_FILE)

    info = qdrant.get_collection(collection)
    print(f"导入完成，Collection 状态：{info.status}，向量数：{info.points_count}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--csv", default="hotel_data.csv")
    parser.add_argument("--collection", default="hotels")
    parser.add_argument("--qdrant", default="http://localhost:6333")
    args = parser.parse_args()

    main(args.csv, args.collection, args.qdrant)
