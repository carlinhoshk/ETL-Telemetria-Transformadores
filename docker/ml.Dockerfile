# Python ML service image (stateless stdlib HTTP server).
FROM python:3.11-slim
WORKDIR /app
COPY python/ml_service/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY python/ml_service ./ml_service
ENV PYTHONPATH=/app
EXPOSE 8081
ENTRYPOINT ["python", "-m", "ml_service"]
CMD ["--host", "0.0.0.0", "--port", "8081"]
