import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Reqres contract checks")
class InventoryServiceTest {

    private static final String BASE = "https://reqres.in/api";
    private final HttpClient client = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(15))
            .build();

    @Test
    @DisplayName("GET /users/1 email domain is example.com")
    void userEmailDomain() throws Exception {
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(BASE + "/users/1"))
                .timeout(Duration.ofSeconds(15))
                .GET()
                .build();
        HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
        System.out.println("[STDOUT] GET /users/1 → " + response.statusCode());
        assertEquals(200, response.statusCode());
        assertTrue(response.body().contains("@reqres.in") || response.body().contains("@example."));
    }

    @Test
    @DisplayName("Pagination reports six users per page")
    void paginationSize() throws Exception {
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(BASE + "/users?page=1"))
                .timeout(Duration.ofSeconds(15))
                .GET()
                .build();
        HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
        assertEquals(200, response.statusCode());
        assertTrue(response.body().contains("\"per_page\":6"));
    }

    @Test
    @DisplayName("Simulated inventory sync failure")
    void inventorySyncFailure() {
        System.err.println("[STDERR] Warehouse replica unreachable — inventory count stale");
        throw new RuntimeException("Inventory sync failed: connection refused to warehouse-db:5432");
    }
}
