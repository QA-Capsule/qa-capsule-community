import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Reqres Users API")
class UserServiceTest {

    private static final String BASE = "https://reqres.in/api";
    private final HttpClient client = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(15))
            .build();

    private HttpResponse<String> get(String path) throws Exception {
        HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(BASE + path))
                .timeout(Duration.ofSeconds(15))
                .GET()
                .build();
        return client.send(request, HttpResponse.BodyHandlers.ofString());
    }

    @Test
    @DisplayName("GET /users?page=1 returns 200")
    void listUsersReturns200() throws Exception {
        HttpResponse<String> response = get("/users?page=1");
        System.out.println("[STDOUT] GET /users?page=1 → " + response.statusCode());
        assertEquals(200, response.statusCode());
        assertTrue(response.body().contains("\"data\""));
    }

    @Test
    @DisplayName("GET /users/2 returns profile with id 2")
    void getUserByIdReturns200() throws Exception {
        HttpResponse<String> response = get("/users/2");
        System.out.println("[STDOUT] GET /users/2 → " + response.statusCode());
        assertEquals(200, response.statusCode());
        assertTrue(response.body().contains("\"id\":2"));
    }

    @Test
    @DisplayName("GET /users/23 wrong status expectation (self-healing demo)")
    void missingUserWrongStatusExpectation() throws Exception {
        HttpResponse<String> response = get("/users/23");
        System.err.println("[STDERR] GET /users/23 → " + response.statusCode() + " but client expects 200");
        assertEquals(200, response.statusCode(), "Contract drift: endpoint returns 404 for unknown user");
    }
}
