# dbt (silver/gold) runner image.
FROM python:3.11-slim
RUN pip install --no-cache-dir dbt-core==1.12.0 dbt-postgres==1.11.0
WORKDIR /work
COPY dbt ./dbt
WORKDIR /work/dbt
ENTRYPOINT []