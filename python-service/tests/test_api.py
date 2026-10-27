"""Test HTTP in-process dengan TestClient (tanpa server sungguhan)."""

import pytest
from fastapi.testclient import TestClient

from app.main import create_app


@pytest.fixture
def client():
    with TestClient(create_app()) as c:  # `with` menjalankan lifespan (seed store)
        yield c


def test_stats(client: TestClient) -> None:
    r = client.post("/api/json", json={"numbers": [1, 2, 3, 4.5, -2]})
    assert r.status_code == 200
    assert r.json() == {"count": 5, "sum": 8.5, "min": -2.0, "max": 4.5, "mean": 1.7, "lang": "python"}


@pytest.mark.parametrize(
    "body",
    [{"numbers": []}, {"numbers": "x"}, {"numbers": ["1"]}, {}],
)
def test_stats_invalid(client: TestClient, body: dict) -> None:
    r = client.post("/api/json", json=body)
    assert r.status_code == 400
    assert "error" in r.json()


def test_items(client: TestClient) -> None:
    assert client.get("/api/items/7").json() == {"id": 7, "name": "item-7", "price": 10.5}
    assert client.get("/api/items/999999").status_code == 404
    assert client.get("/api/items/abc").status_code == 400
    assert client.put("/api/items/5", json={"name": "x"}).status_code == 400
    assert client.put("/api/items/5", json={"name": " ", "price": 1}).status_code == 400
    assert client.put("/api/items/5", json={"name": "x", "price": -1}).status_code == 400
    r = client.put("/api/items/5", json={"name": "x", "price": 2})
    assert r.json() == {"id": 5, "name": "x", "price": 2.0}


def test_cpu(client: TestClient) -> None:
    assert client.get("/api/cpu?n=20").json()["result"] == 6765
    assert client.get("/api/cpu?n=99").status_code == 400
    assert client.get("/api/cpu?n=abc").status_code == 400
