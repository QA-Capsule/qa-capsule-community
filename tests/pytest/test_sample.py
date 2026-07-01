"""
Live API tests against reqres.in (public REST demo).

Most tests perform real HTTP calls. Failures in the last class are intentional
so QA Capsule ingest + MCP healing gate receive realistic API incidents.
"""

import sys

import pytest
import requests

BASE_URL = "https://reqres.in/api"
TIMEOUT = 15


class TestReqresUsers:
    def test_list_users_page_one(self):
        response = requests.get(f"{BASE_URL}/users", params={"page": 1}, timeout=TIMEOUT)
        print(f"[STDOUT] GET /users?page=1 → {response.status_code}")
        assert response.status_code == 200
        body = response.json()
        assert "data" in body and len(body["data"]) > 0

    def test_get_user_two_by_id(self):
        response = requests.get(f"{BASE_URL}/users/2", timeout=TIMEOUT)
        print(f"[STDOUT] GET /users/2 → {response.status_code}")
        assert response.status_code == 200
        assert response.json()["data"]["id"] == 2
        assert "@" in response.json()["data"]["email"]

    def test_create_user_returns_201(self):
        payload = {"name": "QA Capsule", "job": "SRE Control Plane"}
        response = requests.post(f"{BASE_URL}/users", json=payload, timeout=TIMEOUT)
        print(f"[STDOUT] POST /users → {response.status_code}")
        assert response.status_code == 201
        assert response.json()["name"] == payload["name"]


class TestReqresIntentionalFailures:
    def test_missing_user_wrong_status_expectation(self):
        response = requests.get(f"{BASE_URL}/users/23", timeout=TIMEOUT)
        print(f"[STDOUT] GET /users/23 → {response.status_code} (expected 404)")
        sys.stderr.write("[STDERR] Contract drift: client still expects HTTP 200\n")
        assert response.status_code == 200

    def test_wrong_user_id_in_response(self):
        response = requests.get(f"{BASE_URL}/users/2", timeout=TIMEOUT)
        assert response.status_code == 200
        assert response.json()["data"]["id"] == 99, "Stale cache returned wrong user id"

    def test_auth_token_missing(self):
        sys.stderr.write("[STDERR] Authorization header missing on protected route\n")
        pytest.fail("Simulated 401 — bearer token not attached to downstream call")
