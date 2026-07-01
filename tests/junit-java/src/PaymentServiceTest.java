import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Reqres Posts API")
class PaymentServiceTest {

    private static final String BASE = "https://reqres.in/api";
    private final HttpClient client = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(15))
            .build();

    @Test
    @DisplayName("POST /users creates resource with 201")
    void createUserReturns201() throws Exception {
        String body = "{\"name\":\"QA Capsule\",\"job\":\"FinOps\"}";
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(BASE + "/users"))
                .timeout(Duration.ofSeconds(15))
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(body))
                .build();
        HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
        System.out.println("[STDOUT] POST /users → " + response.statusCode());
        assertEquals(201, response.statusCode());
        assertTrue(response.body().contains("QA Capsule"));
    }

    @Test
    @DisplayName("GET /unknown returns 404")
    void unknownRouteReturns404() throws Exception {
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(BASE + "/unknown-route"))
                .timeout(Duration.ofSeconds(15))
                .GET()
                .build();
        HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
        assertEquals(404, response.statusCode());
    }

    @Test
    @DisplayName("POST body missing job field (self-healing demo)")
    void createUserMissingJobField() throws Exception {
        String body = "{\"name\":\"Incomplete\"}";
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(BASE + "/users"))
                .timeout(Duration.ofSeconds(15))
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(body))
                .build();
        HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
        System.err.println("[STDERR] Response missing job field: " + response.body());
        assertTrue(response.body().contains("\"job\":\"Engineer\""), "Downstream schema changed — job default missing");
    }
}
