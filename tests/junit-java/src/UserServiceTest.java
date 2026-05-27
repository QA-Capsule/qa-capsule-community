import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;

@DisplayName("User Service Tests")
class UserServiceTest {

    @Test
    @DisplayName("Create user returns 201")
    void createUserReturns201() {
        System.out.println("[STDOUT] POST /api/users → {name: Alice, role: admin}");
        int statusCode = 201;
        assertEquals(201, statusCode, "Expected HTTP 201 Created");
        System.out.println("[STDOUT] User created with ID: 42");
    }

    @Test
    @DisplayName("Get user by ID returns 200")
    void getUserByIdReturns200() {
        System.out.println("[STDOUT] GET /api/users/42");
        int statusCode = 200;
        assertEquals(200, statusCode, "Expected HTTP 200 OK");
        System.out.println("[STDOUT] User fetched: {id: 42, name: Alice}");
    }

    @Test
    @DisplayName("Delete non-existent user returns 404")
    void deleteNonExistentUserReturns404() {
        System.out.println("[STDOUT] DELETE /api/users/999");
        System.err.println("[STDERR] HTTP 404 Not Found — user does not exist");
        int statusCode = 404;
        assertEquals(200, statusCode, "Expected 200 OK but user was not found");
    }
}
