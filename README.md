# hotel-rag

Hotel occupancy Q&A demo — Python data generation + Go HTTP service + LLM.

Supports two retrieval modes: keyword search (default) and semantic vector search via Qdrant + Jina.

## Stack

| Layer | Tech |
|---|---|
| Data | Python |
| Retrieval | Keyword (default) · Qdrant + Jina (optional) |
| Service | Go |
| LLM | Claude / DeepSeek / Ollama |

## Quick Start

```bash
# generate data
python scripts/gen_data.py

# configure
cp config.example.yaml config.yaml  # set llm.provider + api_key

# (optional) enable vector search: start Qdrant, then
JINA_API_KEY=xxx python scripts/gen_embeddings.py
# set qdrant.jina_api_key in config.yaml

# run
go mod tidy && go run cmd/server/main.go
# → http://localhost:8080
```

## Keyword vs Vector Search

Same query: *"靠近地铁的酒店"*

| Keyword | Vector (Qdrant + Jina) |
|---|---|
| ![keyword](docs/screenshots/demo.png) | ![vector](docs/screenshots/demo_vector.png) |

Keyword search has no "metro" field to match against, so it falls back to a vague district-level answer. Vector search retrieves semantically relevant records and returns a concrete hotel list.
