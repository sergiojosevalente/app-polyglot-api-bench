# ---- build stage: install dependency ke virtualenv ----
FROM python:3.13-slim AS builder
ENV PIP_NO_CACHE_DIR=1 PIP_DISABLE_PIP_VERSION_CHECK=1
RUN python -m venv /opt/venv
ENV PATH=/opt/venv/bin:$PATH
COPY requirements.txt .
RUN pip install -r requirements.txt

# ---- runtime stage: interpreter Python + venv + kode (tetap perlu runtime, seperti PHP/Node) ----
FROM python:3.13-slim
COPY --from=builder /opt/venv /opt/venv
ENV PATH=/opt/venv/bin:$PATH \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1
WORKDIR /app
COPY app ./app
RUN useradd --system --no-create-home app
USER app
EXPOSE 8080
# 1 worker = 1 proses = 1 event loop (dibatasi GIL). Menambah --workers memakai lebih
# banyak core, tetapi setiap proses punya store sendiri (sama seperti Node cluster).
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080", \
     "--workers", "1", "--loop", "uvloop", "--http", "httptools", \
     "--no-access-log", "--timeout-keep-alive", "65"]
