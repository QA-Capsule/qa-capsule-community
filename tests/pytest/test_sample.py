import sys
import time


class TestAuthenticationSuite:
    def test_login_with_valid_credentials(self):
        print("[STDOUT] POST /api/auth/login → {'email': 'user@example.com'}")
        token = "eyJhbGciOiJIUzI1NiJ9.valid"
        assert token.startswith("eyJ"), "Expected a valid JWT token"
        print(f"[STDOUT] Auth token issued: {token[:20]}...")

    def test_login_with_invalid_password(self):
        print("[STDOUT] POST /api/auth/login → {'email': 'user@example.com', 'password': 'wrong'}")
        sys.stderr.write("[STDERR] 401 Unauthorized — credentials rejected\n")
        status_code = 401
        assert status_code == 200, f"Expected 200 OK, got {status_code}"

    def test_token_expiry_raises_401(self):
        print("[STDOUT] GET /api/profile with expired token")
        sys.stderr.write("[STDERR] Token expired — server returned 401\n")
        expired = True
        assert not expired, "Token should still be valid but it expired prematurely"


class TestAPIValidationSuite:
    def test_create_user_returns_201(self):
        print("[STDOUT] POST /api/users → {'name': 'Alice', 'role': 'admin'}")
        status_code = 201
        assert status_code == 201
        print("[STDOUT] User created successfully with ID: 42")

    def test_payload_missing_required_field(self):
        print("[STDOUT] POST /api/users → {} (empty payload)")
        sys.stderr.write("[STDERR] HTTP 422 Unprocessable Entity: 'name' is required\n")
        status_code = 422
        assert status_code == 201, f"Expected 201 Created, got {status_code}"

    def test_rate_limit_triggers_429(self):
        print("[STDOUT] Sending 150 requests in 1 second to /api/search")
        sys.stderr.write("[STDERR] HTTP 429 Too Many Requests — rate limit exceeded\n")
        status_code = 429
        assert status_code == 200, f"Rate limiter triggered unexpectedly: {status_code}"


class TestPerformanceSuite:
    def test_response_time_under_threshold(self):
        print("[STDOUT] GET /api/products → measuring response time")
        simulated_ms = 180
        print(f"[STDOUT] Response received in {simulated_ms}ms")
        assert simulated_ms < 500, f"Response too slow: {simulated_ms}ms"

    def test_database_query_timeout(self):
        print("[STDOUT] Running complex JOIN query across 3 tables")
        sys.stderr.write("[STDERR] Query timeout after 30s — no response from replica\n")
        raise TimeoutError("Database query exceeded the 30s SLA threshold")

    def test_cache_hit_returns_instantly(self):
        print("[STDOUT] GET /api/products/42 (cache expected)")
        time.sleep(0.01)
        cache_hit = True
        assert cache_hit, "Expected cache HIT but got MISS — Redis may be down"
        print("[STDOUT] Cache HIT confirmed — served in <10ms")
