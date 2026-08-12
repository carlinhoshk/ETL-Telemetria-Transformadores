"""Build the portfolio notebooks from markdown/code cell specs.

Not a runtime dependency: regenerates notebooks/*.ipynb deterministically.
Usage:  .venv/bin/python notebooks/build_notebooks.py
"""
from __future__ import annotations

from pathlib import Path

import nbformat
from nbformat.v4 import new_code_cell, new_markdown_cell, new_notebook

REPO = Path(__file__).resolve().parent.parent
NOTEBOOKS = Path(__file__).resolve().parent

# Bootstrap injected at the top of every notebook first code cell. Robust to
# CWD because it resolves paths from this module's location.
BOOTSTRAP = (
    "import sys\n"
    f"sys.path.insert(0, r'{REPO}')\n"
    f"sys.path.insert(0, r'{NOTEBOOKS}')\n"
    "import common\n"
)

BASE = (
    "> **Pré-requisito:** banco local com dados (`make db && make migrate && make smoke`)\n"
    "> e o ML service rodando (`make ml-run &`) — ou apenas `make demo`.\n"
    ">\n"
    "> Carregar o helper compartilhado (bootstrap de imports, leitores de dados,\n"
    "> URLs dos serviços) da primeira célula. `notebooks/common.py`.\n"
)

DEFS: list[tuple[str, str, list[tuple[str, str]]]] = [
    (
        "01_historical_base.ipynb",
        "Requisito 1 — Estruturar e preparar bases históricas de projetos de transformadores",
        [
            (
                "md",
                "# 1. Base histórica de projetos de transformadores\n\n"
                "**Requisito da vaga:** *Estruturar e preparar bases históricas de projetos de "
                "transformadores*.\n\n"
                "Neste notebook carregamos a base de **projetos de engenharia** (frota sintética de "
                "40 transformadores), examinamos distribuições, validamos regras de domínio e "
                "preparamos a base que alimenta o mecanismo de similaridade (Requisito 4).\n\n"
                "Fonte: `dbt/seeds/transformers.csv` (gerada por `make seed`).",
            ),
            ("md", BASE),
            (
                "code",
                BOOTSTRAP
                + "import pandas as pd\n"
                "pd.set_option('display.max_columns', None)\n"
                "\n"
                "design = common.load_design_base()\n"
                "print(f'{design.shape[0]} projetos, {design.shape[1]} atributos')",
            ),
            (
                "code",
                "design.head()",
            ),
            (
                "md",
                "## Inventário e tipologia\n\n"
                "Como o escopo cobre linhas de transmissão e distribuição/industrial, olhamos os "
                "atributos qualitativos: aplicação, vetor de ligação e regime de resfriamento.",
            ),
            (
                "code",
                "design.groupby(['application', 'cooling_type']).size().unstack(fill_value=0)",
            ),
            (
                "code",
                "design.groupby('application')['rated_power_mva'].describe().round(1)",
            ),
            (
                "md",
                "## Distribuições das variáveis de projeto\n\n"
                "Potência, tensões e ano de comissionamento guiam a faixa de similaridade.",
            ),
            (
                "code",
                "import matplotlib.pyplot as plt\n"
                "fig, axes = plt.subplots(1, 3, figsize=(13, 3.5))\n"
                "design['rated_power_mva'].hist(ax=axes[0]); axes[0].set_title('Potência (MVA)')\n"
                "design['hv_voltage_kv'].hist(ax=axes[1]); axes[1].set_title('HV (kV)')\n"
                "design['commissioning_year'].hist(ax=axes[2]); axes[2].set_title('Ano de comissionamento')\n"
                "plt.tight_layout()\n"
                "plt.show()",
            ),
            (
                "md",
                "## Validação das regras de domínio\n\n"
                "A camada de ingestão valida faixas físicas e padrões (o mesmo código de `internal/domain` "
                "em Go). Reproduzimos aqui os checks essenciais para garantir que a base histórica é limpa.",
            ),
            (
                "code",
                "checks = {\n"
                "    'ids únicos': design['transformer_id'].is_unique,\n"
                "    'potência > 0': (design['rated_power_mva'] > 0).all(),\n"
                "    'HV > LV': (design['hv_voltage_kv'] > design['lv_voltage_kv']).all(),\n"
                "    'frequência 50/60 Hz': design['frequency_hz'].isin([50, 60]).all(),\n"
                "    'fases 1 ou 3': design['phase_count'].isin([1, 3]).all(),\n"
                "    'impedância > 0': (design['impedance_percent'] > 0).all(),\n"
                "    'dimensões > 0': ((design[[c for c in design if c.endswith('_m')]] > 0)\n"
                "                        .all().all()),\n"
                "}\n"
                "pd.Series(checks, name='ok')",
            ),
            (
                "code",
                "# Preparação: reordenar colunas numéricas usadas pela similaridade (ordem estável).\n"
                "feature_cols = common.design_features()\n"
                "base_matrix = design[['transformer_id'] + feature_cols].copy()\n"
                "base_matrix.shape",
            ),
            (
                "md",
                "## Conclusão\n\n"
                "- A base histórica está estruturada e pronta para consumo analítico e pelo mecanismo de "
                "similaridade.\n"
                "- Colunas numéricas de projeto em ordem estável (`docs/similarity.md`).\n"
                "- Tipologia condizente com a frota sintética (transmissão e distribuição/industrial).",
            ),
        ],
    ),
    (
        "02_sql_pipeline.ipynb",
        "Requisito 2 — Desenvolver consultas SQL e pipelines de dados",
        [
            (
                "md",
                "# 2. Consultas SQL e pipeline de dados (medalhão)\n\n"
                "**Requisito da vaga:** *Desenvolver consultas SQL e pipelines de dados*.\n\n"
                "Percorremos o modelo medalhão do projeto:\n\n"
                "1. **Bronze** — `raw_telemetry` (payload original, JSONB) e `raw_events`.\n"
                "2. **Silver** — `stg_telemetry` (validada/deduplicada) e `int_telemetry` "
                "(campos derivados: `thermal_stress_index`).\n"
                "3. **Gold** — star schema: `dim_time`, `dim_transformer`, `dim_location`, "
                "`dim_sensor` + `fact_transformer_measurement`.\n\n"
                "Pipeline ELT orquestrado por **dbt** (`dbt/models/silver`, `dbt/models/gold`).",
            ),
            ("md", BASE),
            (
                "code",
                BOOTSTRAP
                + "import pandas as pd\n"
                "pd.set_option('display.max_columns', None)",
            ),
            (
                "md",
                "## Bronze → tabelas operacionais\n\n"
                "A ingestão (Go) grava o payload original em `raw_telemetry` e a medição normalizada "
                "em `measurements` (deduplicada por `{transformer_id}@{timestamp}`).",
            ),
            (
                "code",
                "rows = common.pg_df('''\n"
                "SELECT transformer_id, count(*) AS n,\n"
                "       min(ts) AS first_ts, max(ts) AS last_ts\n"
                "FROM measurements GROUP BY transformer_id ORDER BY n DESC LIMIT 5\n"
                "''')\n"
                "rows",
            ),
            (
                "md",
                "## Silver — validação e campos derivados\n\n"
                "`int_telemetry` adiciona `thermal_stress_index` e margens de temperatura, e "
                "recomputa o estado do transformador independentemente do emissor.",
            ),
            (
                "code",
                "common.pg_df('''\n"
                "SELECT transformer_id, state_recomputed, count(*) AS n,\n"
                "       round(avg(thermal_stress_index)::numeric, 4) AS avg_tsi\n"
                "FROM int_telemetry GROUP BY 1,2 ORDER BY n DESC LIMIT 5\n"
                "''')",
            ),
            (
                "md",
                "## Gold — star schema analítico\n\n"
                "Join entre fato e dimensões para análise por transformador/aplicação.",
            ),
            (
                "code",
                "common.pg_df('''\n"
                "SELECT dt.transformer_key, dt.application, count(*) AS n_measurements,\n"
                "       round(avg(fm.oil_temperature_c)::numeric, 2) AS avg_oil_temp_c\n"
                "FROM fact_transformer_measurement fm\n"
                "JOIN dim_transformer dt   ON fm.transformer_key   = dt.transformer_key\n"
                "GROUP BY 1,2 ORDER BY n_measurements DESC LIMIT 6\n"
                "''')",
            ),
            (
                "md",
                "## Dril-down de negócio\n\n"
                "Consulta analítica: perfil térmico médio por aplicação (região/segmento), "
                "reaproveitando `dim_location`.",
            ),
            (
                "code",
                "common.pg_df('''\n"
                "SELECT dl.region, dt.application,\n"
                "       round(avg(fm.oil_temperature_c)::numeric, 2) AS avg_oil_temp_c,\n"
                "       count(*) AS n\n"
                "FROM fact_transformer_measurement fm\n"
                "JOIN dim_transformer dt ON fm.transformer_key = dt.transformer_key\n"
                "JOIN dim_location dl    ON fm.location_key    = dl.location_key\n"
                "GROUP BY 1,2 ORDER BY 1,2\n"
                "''')",
            ),
            (
                "md",
                "## Conclusão\n\n"
                "- Modelo medalhão bronze→silver→gold com dbt reproduzível (`make dbt`), testes de dados "
                "(20 em silver, 25 em gold).\n"
                "- Consultas analíticas diretas em SQL sobre o star schema — pronto para dashboards.",
            ),
        ],
    ),
    (
        "03_integrations.ipynb",
        "Requisito 3 — Integrações entre banco de dados, APIs e plataformas internas",
        [
            (
                "md",
                "# 3. Integrações: PostgreSQL, API Go e serviço ML\n\n"
                "**Requisito da vaga:** *Implementar integrações entre banco de dados, APIs e "
                "plataformas internas*.\n\n"
                "Demonstramos as integrações reais do projeto percorrendo o arco de ponta a ponta:\n\n"
                "```\nMqtt (simulador) → ingestion (Go) → PostgreSQL → ML service (Python) → API (Go)\n```\n\n"
                "- **Banco:** leitura direta via `psycopg2` (bronze/operacional).\n"
                "- **Serviço ML:** HTTP `POST /similar` (contrato JSON, `docs/ml-service.md`).\n"
                "- **API Go:** HTTP `GET /transformers`, `/{id}/statistics`, `/{id}/telemetry`.\n\n"
                "Tudo roda localmente com `make demo`.",
            ),
            ("md", BASE),
            (
                "code",
                BOOTSTRAP
                + "import requests\n"
                "import pandas as pd\n"
                "pd.set_option('display.max_columns', None)\n"
                "print('ML  :', common.ML_URL)\n"
                "print('API :', common.API_URL)",
            ),
            (
                "md",
                "## 3.1 PostgreSQL → análise\n\n"
                "Consulta direta ao modelo operacional (a mesma base que a API serve).",
            ),
            (
                "code",
                "transformers = common.pg_df('''\n"
                "SELECT transformer_id, rated_power_mva, hv_voltage_kv, application\n"
                "FROM transformers ORDER BY transformer_id LIMIT 5\n"
                "''')\n"
                "transformers",
            ),
            (
                "md",
                "## 3.2 API Go → contratos HTTP\n\n"
                "A API Go (`internal/api`) expõe os dados operacionais. Consumimos os endpoints reais.",
            ),
            (
                "code",
                "import json\n"
                "r = requests.get(f'{common.API_URL}/transformers/TR-001', timeout=10)\n"
                "r.raise_for_status()\n"
                "json.dumps(r.json(), indent=2, ensure_ascii=False)",
            ),
            (
                "code",
                "r = requests.get(f'{common.API_URL}/transformers/TR-001/statistics', timeout=10)\n"
                "r.raise_for_status()\n"
                "json.dumps(r.json(), indent=2)",
            ),
            (
                "md",
                "## 3.3 Python ML → serviço de IA\n\n"
                "O mecanismo de similaridade roda no **serviço Python ML** (independência de plataforma): "
                "a API Go delega para ele. Consumimos a mesma rota.",
            ),
            (
                "code",
                "matches = common.similar_for('TR-001', top_k=5)\n"
                "matches",
            ),
            (
                "code",
                "# Prova do fluxo completo: ID alvo -> API Go -> ML service -> top-k com score.\n"
                "target = 'TR-001'\n"
                "api = requests.get(f'{common.API_URL}/transformers/{target}', timeout=10).json()\n"
                "ml  = requests.post(f'{common.ML_URL}/similar', timeout=15, json={\n"
                "        'target': api,\n"
                "        'candidates': common.to_plain_records(\n"
                "            common.pg_df('''SELECT * FROM transformers\n"
                "                        WHERE transformer_id <> %s''', (target,))),\n"
                "        'top_k': 5}).json()\n"
                "pd.DataFrame(ml['results'])",
            ),
            (
                "md",
                "## Conclusão\n\n"
                "- Integrações banco ↔ API ↔ serviço ML funcionam de ponta a ponta e consumidas via "
                "contratos JSON estáveis.\n"
                "- Mesma chave (`transformer_id`) atravessa todas as camadas — sem conversão ad-hoc.",
            ),
        ],
    ),
    (
        "04_similarity.ipynb",
        "Requisito 4 — Apoiar o desenvolvimento do mecanismo de similaridade entre projetos",
        [
            (
                "md",
                "# 4. Mecanismo de similaridade entre projetos\n\n"
                "**Requisito da vaga:** *Apoiar o desenvolvimento do mecanismo de similaridade entre "
                "projetos*.\n\n"
                "Baseline sem LLM: distância **Euclidiana normalizada** sobre os vetores de *features de "
                "projeto* escalados (`StandardScaler`). A candidata mais próxima no espaço padronizado "
                "recebe score mais alto em `[0, 1]`.\n\n"
                "Implementação real: `python/ml_service/features.py` + `similarity.py`. "
                "Metodologia em `docs/similarity.md`.",
            ),
            ("md", BASE),
            (
                "code",
                BOOTSTRAP
                + "import pandas as pd\n"
                "import numpy as np\n"
                "pd.set_option('display.max_columns', None)\n"
                "from ml_service.features import fit_design_features, DESIGN_FEATURES\n"
                "from ml_service.similarity import similarity_scores",
            ),
            (
                "md",
                "## Features de projeto (ordem estável)\n\n"
                "As mesmas colunas numéricas da base histórica (Requisito 1) e do ELT (Requisito 2).",
            ),
            (
                "code",
                "design = common.load_design_base()\n"
                "feature_cols = common.design_features()\n"
                "display(pd.DataFrame({'feature': feature_cols}))",
            ),
            (
                "md",
                "## Escalamento\n\n"
                "Antes da distância, cada dimensão é padronizada (média 0, desvio 1). Assim `rated_power_mva` "
                "no padrão MVA e `commissioning_year` competem em escala equivalente.",
            ),
            (
                "code",
                "design_records = [r.to_dict() for _, r in design.iterrows()]\n"
                "model = fit_design_features(design_records)\n"
                "pv = model.transform(design_records[0])\n"
                "pd.DataFrame({'feature': model.columns, 'scaled_component': np.round(pv, 3)})",
            ),
            (
                "md",
                "## Similaridade computada localmente (mesmo algoritmo do serviço)\n\n"
                "Alvo: **TR-001**. Identificamos os candidatos mais próximos e os scores; o alvo é "
                "excluído dos candidatos (sem auto-match).",
            ),
            (
                "code",
                "target_id = 'TR-001'\n"
                "candidates = [r for r in design_records if r['transformer_id'] != target_id]\n"
                "scores = similarity_scores(design_records[0], candidates, model)\n"
                "pd.DataFrame(scores[:6], columns=['transformer_id', 'score'])",
            ),
            (
                "md",
                "## Comparando com o serviço ML\n\n"
                "Confere se o resultado local bate com `POST /similar` (fora do processo Python — o "
                "MESMO código roda no serviço).",
            ),
            (
                "code",
                "local = dict(similarity_scores(design_records[0], candidates, model)[:5])\n"
                "remote = common.similar_for(target_id, top_k=5).set_index('transformer_id')['score'].to_dict()\n"
                "pd.DataFrame({'local': local, 'servico_ml': remote}).round(4)",
            ),
            (
                "md",
                "## Baseline vs. distância bruta\n\n"
                "O score `1/(1+distância)` é monótono: quanto menor a distância, maior o score — "
                "legível e comparável entre alvos.",
            ),
            (
                "code",
                "raw = [(rid, np.linalg.norm(model.transform(design_records[0]) - model.transform(r)))\n"
                "       for rid, r in [(c['transformer_id'], c) for c in candidates[:5]]]\n"
                "pd.DataFrame(raw, columns=['transformer_id', 'distancia_bruta']).assign(\n"
                "    score=lambda d: (1 / (1 + d['distancia_bruta'])).round(4))",
            ),
            (
                "md",
                "## Conclusão\n\n"
                "- Mecanismo baseline rápido, determinístico e sem LLM — adequado para embasar propostas "
                "e reuso de engenharia.\n"
                "- Resultados do serviço Python e do cálculo local coincidem (mesmo código).\n"
                "- Evolução possível: embeddings específicos de domínio ou similaridade multi-critério "
                "(documentado como próximo passo, fora de escopo).",
            ),
        ],
    ),
    (
        "05_ml_services.ipynb",
        "Requisito 5 — Desenvolver serviços para disponibilização dos modelos de IA",
        [
            (
                "md",
                "# 5. Serviços para disponibilização dos modelos de IA\n\n"
                "**Requisito da vaga:** *Desenvolver serviços para disponibilização dos modelos de IA*.\n\n"
                "Os modelos (similaridade e anomalia) são **expostos como serviço HTTP** — `POST /similar` "
                "e `POST /anomaly` — consumidos pela API Go e por este notebook. Stateless, pronto para "
                "escalar em contêineres (Azure Container Apps, Phase 16).\n\n"
                "Implementação real: `python/ml_service/service.py`.",
            ),
            ("md", BASE),
            (
                "code",
                BOOTSTRAP
                + "import requests\n"
                "import pandas as pd\n"
                "pd.set_option('display.max_columns', None)\n"
                "print('ML service em', common.ML_URL)",
            ),
            (
                "md",
                "## Serviço 1 — Similaridade (`POST /similar`)\n\n"
                "Entrada: `target` + `candidates` (projetos). Saída: top-k com score.",
            ),
            (
                "code",
                "r = requests.get(f'{common.API_URL}/transformers/TR-001', timeout=10)\n"
                "target = r.json()\n"
                "candidates = common.to_plain_records(common.pg_df('SELECT * FROM transformers'))\n"
                "resp = requests.post(f'{common.ML_URL}/similar', json={\n"
                "    'target': target, 'candidates': candidates, 'top_k': 5}, timeout=15)\n"
                "resp.raise_for_status()\n"
                "pd.DataFrame(resp.json()['results'])",
            ),
            (
                "md",
                "## Serviço 2 — Anomalia (`POST /anomaly`)\n\n"
                "Entrada: telemetria de um transformador. Saída: predição de outlier por ponto "
                "(`anomaly: 1`) com score. Modelo: `IsolationForest`.",
            ),
            (
                "code",
                "telemetry = common.pg_df('''\n"
                "SELECT transformer_id, ts AS timestamp, load_percent, ambient_temperature_c,\n"
                "       oil_temperature_c, winding_temperature_c, oil_level_percent\n"
                "FROM measurements ORDER BY ts LIMIT 200\n"
                "''')\n"
                "payload = {'measurements': common.to_plain_records(telemetry)}\n"
                "resp = requests.post(f'{common.ML_URL}/anomaly', json=payload, timeout=30)\n"
                "resp.raise_for_status()\n"
                "anomalies = pd.DataFrame(resp.json()['results'])\n"
                "anomalies",
            ),
            (
                "code",
                "print('pontos analisados:', len(anomalies))\n"
                "print('outliers detectados:', int(anomalies['anomaly'].sum()))\n"
                "outliers = anomalies[anomalies['anomaly'] == 1]\n"
                "outliers.head()",
            ),
            (
                "md",
                "## Robustez do serviço\n\n"
                "- Erros de entrada → HTTP 400 com mensagem clara; rota desconhecida → 404.\n"
                "- Stateless: cada chamada refita o scaler — simples de escalar/replicar.",
            ),
            (
                "code",
                "r = requests.post(f'{common.ML_URL}/similar', json={'bad': 'payload'}, timeout=10)\n"
                "print('status:', r.status_code)\n"
                "print(r.json())",
            ),
            (
                "md",
                "## Conclusão\n\n"
                "- Modelos de IA disponibilizados como serviços HTTP versionados e consumíveis pela "
                "API Go — plataforma pronta para orquestração em contêineres.\n"
                "- Caminho para Azure: rotas expostas + healthcheck (`GET /health`) = probes nativos de "
                "Container Apps (docs/azure.md).",
            ),
        ],
    ),
]


def build() -> None:
    out_dir = Path(__file__).resolve().parent
    for fname, title, cells in DEFS:
        nb = new_notebook(
            cells=[new_markdown_cell(c) if kind == "md" else new_code_cell(c) for kind, c in cells],
            metadata={
                "kernelspec": {
                    "display_name": "Python 3",
                    "language": "python",
                    "name": "python3",
                },
                "language_info": {"name": "python", "version": "3.11"},
            },
        )
        path = out_dir / fname
        with open(path, "w", encoding="utf-8") as f:
            nbformat.write(nb, f)
        print(f"wrote {path} — {title}")


if __name__ == "__main__":
    build()