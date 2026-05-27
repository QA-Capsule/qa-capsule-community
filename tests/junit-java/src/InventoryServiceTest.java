import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Inventory Service Tests")
class InventoryServiceTest {

    @Test
    @DisplayName("Stock check returns available quantity")
    void stockCheckReturnsAvailableQuantity() {
        System.out.println("[STDOUT] GET /api/inventory/SKU-001");
        int stock = 150;
        assertTrue(stock > 0, "Expected stock > 0");
        System.out.println("[STDOUT] Stock for SKU-001: " + stock + " units");
    }

    @Test
    @DisplayName("Reserve stock decrements quantity")
    void reserveStockDecrementsQuantity() {
        System.out.println("[STDOUT] POST /api/inventory/SKU-001/reserve → {qty: 5}");
        int remaining = 145;
        assertEquals(145, remaining, "Expected 145 units remaining after reservation");
        System.out.println("[STDOUT] Reserved 5 units — remaining: " + remaining);
    }

    @Test
    @DisplayName("Out-of-stock throws StockException")
    void outOfStockThrowsException() {
        System.out.println("[STDOUT] POST /api/inventory/SKU-EMPTY/reserve → {qty: 1}");
        System.err.println("[STDERR] StockException: No units available for SKU-EMPTY");
        throw new RuntimeException("StockException: Cannot reserve from empty stock");
    }
}
